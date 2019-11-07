// Package main is the Reset Pipeline used with CodeBuild under the Reset
// CodePipeline for an account
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"text/template"

	"github.com/pkg/errors"

	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/reset"
	"github.com/avast/retry-go"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/rds"
)

// main will run through the reset process for an account which involves using
// aws-nuke
func main() {
	// Initialize a service container
	svc := &service{}
	config := svc.config()

	if !config.isNukeEnabled {
		log.Println("INFO: Nuke is set in Dry Run mode and will not remove " +
			"any resources and cannot set back the state of the DCE child account " +
			"Please set 'RESET_NUKE_DRY_RUN' to not 'true' to exit Dry Run " +
			"mode.")
	}

	// Delete RDS automated backups
	if config.isNukeEnabled {
		log.Println("RDS backup nuke")
		awsSession := svc.awsSession()
		tokenService := svc.tokenService()
		roleArn := "arn:aws:iam::" + config.accountID + ":role/" + config.accountAdminRoleName
		rdsCreds := tokenService.NewCredentials(awsSession, roleArn)
		rdsSession, err := tokenService.NewSession(awsSession, roleArn)
		if err != nil {
			log.Fatalf("Failed to create rds session %s: %s\n", config.accountID, err)
		}
		rdsClient := rds.New(rdsSession, &aws.Config{
			Credentials: rdsCreds,
		})
		rdsReset := reset.RdsReset{
			Client: rdsClient,
		}
		err = reset.DeleteRdsBackups(rdsReset)
		if err != nil {
			log.Fatalf("Failed to execute aws-nuke RDS backup on account %s: %s\n", config.accountID, err)
		}

		// Delete Athena resources
		log.Println("Starting Athena nuking")

		athenaCreds := tokenService.NewCredentials(awsSession, roleArn)
		athenaClient := athena.New(awsSession, &aws.Config{
			Credentials: athenaCreds,
		})
		athenaReset := &reset.AthenaReset{
			Client: athenaClient,
		}
		err = reset.DeleteAthenaResources(athenaReset)
		if err != nil {
			log.Fatalf("Failed to execute aws-nuke athena on account %s: %s\n", config.accountID, err)
		}
	}

	// Execute aws-nuke, to delete all resources from the account
	err := nukeAccount(
		svc,
		// Execute nuke as a dry run, if isNukeEnabled is off
		config.isNukeEnabled == false,
	)
	if err != nil {
		log.Fatalf("Failed to execute aws-nuke on account %s: %s\n", config.accountID, err)
	}
	log.Printf("%s  :  Nuke Success\n", config.accountID)

	// Update the DB with Account/Lease statuses
	err = updateDBPostReset(svc.db(), svc.snsService(), config.accountID, common.RequireEnv("RESET_COMPLETE_TOPIC_ARN"))
	if err != nil {
		log.Fatalf("Failed to update the DB post-reset for account %s:  %s", config.accountID, err)
	}
}

// updateDBPostReset changes any leases for the Account
// from "Status=ResetLock" to "Status=Active"
// Also, if the account was set as "Status=NotReady",
// will update to "Status=Ready"
func updateDBPostReset(dbSvc db.DBer, snsSvc common.Notificationer, accountID string, snsTopicArn string) error {

	// If the Account.Status=NotReady, change it back to Status=Ready
	log.Printf("Setting Account Status from NotReady to Ready: %s", accountID)
	account, err := dbSvc.TransitionAccountStatus(
		accountID,
		db.NotReady, db.Ready)

	// Ignore StatusTransitionErrors
	// (just means the status was NOT previously NotReady")
	if err != nil {
		if _, ok := err.(*db.StatusTransitionError); !ok {
			return err
		}
		account, err = dbSvc.GetAccount(accountID)
		if err != nil {
			return err
		}
	}

	log.Printf("Notifying Reset Topic that the account is complete for: %s", accountID)
	snsMessage, err := common.PrepareSNSMessageJSON(account)
	if err != nil {
		log.Printf("Failed to create SNS account-created message for %s: %s", accountID, err)
		return err
	}
	log.Print(snsMessage)
	_, err = snsSvc.PublishMessage(aws.String(snsTopicArn), aws.String(snsMessage), true)
	if err != nil {
		log.Print("Issue in publishing message: %s" + err.Error())
		return err
	}
	return nil
}

func nukeAccount(svc *service, isDryRun bool) error {
	// Generate the configuration of the yaml file using the template file
	// provided and substituting necessary phrases.

	config := svc.config()

	// Create the file
	configFile := fmt.Sprintf("/tmp/nuke-config-%s.yml", config.accountID)
	f, err := os.Create(configFile)
	if err != nil {
		log.Fatalf("Failed to create file %s: %s", configFile, err)
		return err
	}
	err = generateNukeConfig(svc, f)
	if err != nil {
		return err
	}

	// Construct Nuke
	nuke := reset.Nuke{}

	// Configure the NukeAccountInput
	nukeAccountInput := reset.NukeAccountInput{
		AccountID:  config.accountID,
		RoleName:   config.accountAdminRoleName,
		ConfigPath: configFile,
		NoDryRun:   !isDryRun,
		Token:      svc.tokenService(),
		Nuke:       nuke,
	}

	// Nukes based on the configuration file that is generated
	// Attempt Nuke 3 times in the case not all resources get deleted
	err = retry.Do(
		func() error {
			return reset.NukeAccount(&nukeAccountInput)
		},
		retry.Attempts(3),         // Retry 3 times
		retry.LastErrorOnly(true), // Only return the last error
	)
	if err != nil {
		return err
	}
	return nil
}

func generateNukeConfig(svc *service, f io.Writer) error {
	config := svc.config()

	// Verify the nuke template configuration to download file from s3 or to
	// use the default
	var templateFile string
	if config.nukeTemplateBucket != "STUB" && config.nukeTemplateKey != "STUB" {
		log.Printf("Using Nuke Configuration from S3: %s/%s",
			config.nukeTemplateBucket, config.nukeTemplateKey)

		// Download the file from S3
		templateFile = fmt.Sprintf("/tmp/nuke-config-template-%s.yml",
			config.accountID)
		err := svc.s3Service().Download(config.nukeTemplateBucket,
			config.nukeTemplateKey, templateFile)
		if err != nil {
			return errors.Wrapf(err, "Failed to download nuke template at s3://%s/%s to %s",
				config.nukeTemplateBucket, config.nukeTemplateKey, templateFile)
		}
	} else {
		log.Printf("Using Default Nuke Configuration: %s",
			config.nukeTemplateDefault)

		// Use default template
		templateFile = config.nukeTemplateDefault
	}

	template, err := template.New(templateFile).ParseFiles(templateFile)
	if err != nil {
		log.Fatalf("Failed to generate nuke config for acount %s using template %s: %s",
			config.accountID, templateFile, err)
		return err
	}

	type templateParams struct {
		ID              string
		AdminRole       string
		PrincipalRole   string
		PrincipalPolicy string
		Regions         []string
	}

	err = template.ExecuteTemplate(f, templateFile, &templateParams{
		ID:              config.accountID,
		AdminRole:       config.accountAdminRoleName,
		PrincipalRole:   config.accountPrincipalRoleName,
		PrincipalPolicy: config.accountPrincipalPolicyName,
		Regions:         config.allowedRegions,
	})
	if err != nil {
		log.Fatalf("Failed to generate nuke config for acount %s using template %s: %s",
			config.accountID, templateFile, err)
		return err
	}

	return nil
}
