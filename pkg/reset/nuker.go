package reset

import (
	"context"

	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/gotidy/ptr"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ekristen/aws-nuke/v3/pkg/awsutil"
	nukecommon "github.com/ekristen/aws-nuke/v3/pkg/common"
	"github.com/ekristen/aws-nuke/v3/pkg/config"
	"github.com/ekristen/aws-nuke/v3/pkg/nuke"
	_ "github.com/ekristen/aws-nuke/v3/resources"

	libnuke "github.com/ekristen/libnuke/pkg/nuke"
	"github.com/ekristen/libnuke/pkg/registry"
	"github.com/ekristen/libnuke/pkg/scanner"
	"github.com/ekristen/libnuke/pkg/types"
)

type Nuker struct {
	Nuke      *libnuke.Nuke
	Config    *config.Config
	NukeInput *NukeAccountInput

	Account       *awsutil.Account
	Creds         *awsutil.Credentials
	ResourceTypes types.Collection

	Logger *logrus.Logger
	ctx    *context.Context
}

func NewNuker(parsedConfig *config.Config, nukeInput *NukeAccountInput) (*Nuker, error) {
	logger := logrus.StandardLogger()
	params := &libnuke.Parameters{
		Force:          true,
		ForceSleep:     5,
		NoDryRun:       nukeInput.NoDryRun,
		MaxWaitRetries: 100,
	}

	filters, err := parsedConfig.Filters(nukeInput.ChildAccountID)
	if err != nil {
		return nil, err
	}

	n := libnuke.New(params, filters, parsedConfig.Settings)
	n.SetLogger(logger.WithField("component", "libnuke"))
	n.RegisterVersion(nukecommon.AppVersion.String())
	ctx := context.Background()

	nuker := &Nuker{
		Nuke:      n,
		Config:    parsedConfig,
		NukeInput: nukeInput,

		Logger: logger,
		ctx:    &ctx,
	}

	err = nuker.ConfigureCreds()
	if err != nil {
		return nil, err
	}

	account, err := awsutil.NewAccount(nuker.Creds, parsedConfig.CustomEndpoints)
	if err != nil {
		return nil, err
	}
	nuker.Account = account

	nuker.Nuke.RegisterValidateHandler(func() error {
		return parsedConfig.ValidateAccount(nuker.NukeInput.ChildAccountID, nuker.Account.Aliases(), false)
	})

	p := &nuke.Prompt{Parameters: params, Account: account, Logger: logger}
	nuker.Nuke.RegisterPrompt(p.Prompt)
	nuker.RegisterResourceTypes()

	err = nuker.RegisterScanners()
	if err != nil {
		return nil, err
	}

	return nuker, nil
}

func (nuker *Nuker) ConfigureCreds() error {
	// Get the Credentials of the Role to be assumed into for the Nuke
	roleArn := "arn:aws:iam::" + nuker.NukeInput.ChildAccountID + ":role/" + nuker.NukeInput.RoleName
	roleSessionName := "DCENuke" + nuker.NukeInput.ChildAccountID
	assumeRoleInputs := sts.AssumeRoleInput{
		RoleArn:         &roleArn,
		RoleSessionName: &roleSessionName,
	}
	assumeRoleOutput, err := nuker.NukeInput.Token.AssumeRole(
		&assumeRoleInputs,
	)
	if err != nil {
		return errors.Wrapf(err, "Failed to assume role for nuking account %s as %s",
			nuker.NukeInput.ChildAccountID, roleArn)
	}

	// Create a Credentials based on the aws credentials stored
	// under the "default" profile.
	creds := awsutil.Credentials{
		AccessKeyID:     *assumeRoleOutput.Credentials.AccessKeyId,
		SecretAccessKey: *assumeRoleOutput.Credentials.SecretAccessKey,
		SessionToken:    *assumeRoleOutput.Credentials.SessionToken,
	}

	nuker.Creds = &creds

	return nil
}

func (nuker *Nuker) RegisterResourceTypes() {
	// Get any specific account level configuration
	accountConfig := nuker.Config.Accounts[nuker.NukeInput.ChildAccountID]

	// Resolve the resource types to be used for the nuke process based on the parameters, global configuration, and
	// account level configuration.
	resourceTypes := types.ResolveResourceTypes(
		registry.GetNames(),
		[]types.Collection{
			registry.ExpandNames(nuker.Nuke.Parameters.Includes),
			nuker.Config.ResourceTypes.GetIncludes(),
			accountConfig.ResourceTypes.GetIncludes(),
		},
		[]types.Collection{
			registry.ExpandNames(nuker.Nuke.Parameters.Excludes),
			nuker.Config.ResourceTypes.Excludes,
			accountConfig.ResourceTypes.Excludes,
		},
		[]types.Collection{
			registry.ExpandNames(nuker.Nuke.Parameters.Alternatives),
			nuker.Config.ResourceTypes.GetAlternatives(),
			accountConfig.ResourceTypes.GetAlternatives(),
		},
		registry.GetAlternativeResourceTypeMapping(),
	)
	nuker.Nuke.RegisterResourceTypes(nuke.Account, resourceTypes...)
	nuker.ResourceTypes = resourceTypes
}

func (nuker *Nuker) RegisterScanners() error {
	// Register the scanners for each region that is defined in the configuration.
	for _, regionName := range nuker.Config.Regions {
		// Step 1 - Create the region object
		region := nuke.NewRegion(regionName, nuker.Account.ResourceTypeToServiceType, nuker.Account.NewSession, nuker.Account.NewConfig)

		// Step 2 - Create the scannerActual object
		scannerActual := scanner.New(regionName, nuker.ResourceTypes, &nuke.ListerOpts{
			Region:    region,
			AccountID: ptr.String(nuker.NukeInput.ChildAccountID),
			Logger: nuker.Logger.WithFields(logrus.Fields{
				"component": "scanner",
				"region":    regionName,
			}),
		})
		scannerActual.SetLogger(nuker.Logger)

		// Step 3 - Register a mutate function that will be called to modify the lister options for each resource type
		// see pkg/nuke/resource.go for the MutateOpts function. Its purpose is to create the proper session for the
		// proper region.
		err := scannerActual.RegisterMutateOptsFunc(nuke.MutateOpts)
		if err != nil {
			return err
		}

		// Step 4 - Register the scannerActual with the nuke object
		err = nuker.Nuke.RegisterScanner(nuke.Account, scannerActual)
		if err != nil {
			return err
		}
	}
	return nil
}

func (nuker *Nuker) Run() error {
	err := nuker.Nuke.Run(*nuker.ctx)
	return err
}
