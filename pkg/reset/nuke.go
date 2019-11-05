package reset

import (
	"github.com/pkg/errors"
	"time"

	"github.com/Optum/dce/pkg/common"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/rebuy-de/aws-nuke/cmd"
	"github.com/rebuy-de/aws-nuke/pkg/awsutil"
)

// NukeAccountInput is the container used for the TokenService and the
// NukeService to execute a Nuke for an AWS Account
type NukeAccountInput struct {
	AccountID  string
	RoleName   string
	ConfigPath string
	NoDryRun   bool
	Token      common.TokenService
	Nuke       Nuker
}

// NukeAccount directly triggers aws-nuke to be called on the
// configuration file provided, bypassing any manual prompts.
// Returns an error if there's any, else nil.
func NukeAccount(input *NukeAccountInput) error {

	// Create a NukeParameter based on the configuration file
	// path and force to bypass prompts.
	params := cmd.NukeParameters{
		ConfigPath:     input.ConfigPath,
		NoDryRun:       input.NoDryRun,
		ForceSleep:     5,
		MaxWaitRetries: 200,
		Force:          true,
	}

	// Get the Credentials of the Role to be assumed into for the Nuke
	roleArn := "arn:aws:iam::" + input.AccountID + ":role/" + input.RoleName
	roleSessionName := "DCENuke" + input.AccountID
	assumeRoleInputs := sts.AssumeRoleInput{
		RoleArn:         &roleArn,
		RoleSessionName: &roleSessionName,
	}
	assumeRoleOutput, err := input.Token.AssumeRole(
		&assumeRoleInputs,
	)
	if err != nil {
		return errors.Wrapf(err, "Failed to assume role for nuking account %s as %s",
			input.AccountID, roleArn)
	}

	// Create a Credentials based on the aws credentials stored
	// under the "default" profile.
	creds := awsutil.Credentials{
		AccessKeyID:     *assumeRoleOutput.Credentials.AccessKeyId,
		SecretAccessKey: *assumeRoleOutput.Credentials.SecretAccessKey,
		SessionToken:    *assumeRoleOutput.Credentials.SessionToken,
	}

	// Create an Account with the crdentials and a Nuke based
	// on the new Account and NukeParameter
	account, err := input.Nuke.NewAccount(creds)
	if err != nil {
		return errors.Wrapf(err, "Failed to configure account %s for aws-nuke as %s",
			input.AccountID, roleArn)
	}
	nuke := cmd.NewNuke(params, *account)

	// Load in the configuration for aws-nuke into the Nuke
	// struct and execute the deletion process.
	// Timeout after 60 minutes
	// https://github.com/golang/go/wiki/Timeouts
	nuke.Config, err = input.Nuke.Load(nuke.Parameters.ConfigPath)
	if err != nil {
		return errors.Wrapf(err, "Failed to load nuke config at %s", nuke.Parameters.ConfigPath)
	}
	c := make(chan error, 1)
	go func() { c <- input.Nuke.Run(nuke) }()
	select {
	case err := <-c:
		if err != nil {
			return errors.Wrapf(err, "Failed to run nuke for account %s as %s",
				input.AccountID, roleArn)
		}
		return nil
	case <-time.After(time.Minute * 60):
		return errors.New("Nuke Timed Out after 60 minutes")
	}
}
