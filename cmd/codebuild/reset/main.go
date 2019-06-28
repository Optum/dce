// Package main is the Reset Pipeline used with CodeBuild under the Reset
// CodePipeline for an account
package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/reset"
	"github.com/avast/retry-go"
)

// main will run through the reset process for an account which involves using
// aws-nuke and reapplying Launchpad via an API
func main() {
	// Initialize a service container
	svc := &service{}
	config := svc.config()

	if !config.isNukeEnabled {
		log.Println("INFO: Nuke is set in Dry Run mode and will not remove " +
			"any resources and cannot set back the state of Redbox Account " +
			"Please set 'RESET_NUKE_DRY_RUN' to not 'true' to exit Dry Run " +
			"mode.")
	}

	// Execute aws-nuke, to delete all resources from the account
	err := nukeAccount(
		svc,
		// Execute nuke as a dry run, if isNukeEnabled is off
		config.isNukeEnabled == false,
	)
	if err != nil {
		log.Fatalf("%s  :  %s\n", config.accountID, err)
	}
	log.Printf("%s  :  Nuke Success\n", config.accountID)

	// Run Launchpad Setup
	// Recreates a clean TF state file in the account, so Launchpad can run
	// We only do this if aws-nuke ran, and deleted LP's TF state file
	if config.isNukeEnabled {
		log.Printf("Launchpad Setup for %s\n", config.accountID)
		err = svc.launchpadAPI().Setup(config.accountID)
		if err != nil {
			log.Fatalf("%s : Could not Setup Launchpad - %s", config.accountID, err)
		}
		log.Printf("Launchpad Setup Success for %s\n", config.accountID)
	}

	// Initiate Launchpad and wait for final status
	if config.isLaunchpadEnabled {
		// Call LaunchpadAccount
		launchpadAccountInput := reset.LaunchpadAccountInput{
			Launchpad:     svc.launchpadAPI(),
			AccountID:     config.accountID,
			MasterAccount: config.launchpadMasterAccount,
			WaitSeconds:   30,
		}
		err := reset.LaunchpadAccount(&launchpadAccountInput)
		if err != nil {
			log.Fatalf("%s  :  %s\n", config.accountID, err)
		}
	} else {
		log.Println("INFO: Launchpad is set as toggled off and cannot set " +
			" back the state of a Redbox Account.")
	}

	// Update the DB with Account/Assignment statuses
	err = updateDBPostReset(svc.db(), config.accountID)
	if err != nil {
		log.Fatalf("%s  :  %s\n", config.accountID, err)
	}
}

// updateDBPostReset changes any assignments for the Account
// from "Status=ResetLock" to "Status=Active"
// Also, if the account was set as "Status=NotReady",
// will update to "Status=Ready"
func updateDBPostReset(dbSvc db.DBer, accountID string) error {
	// Find any assignment for the Account
	assignments, err := dbSvc.FindAssignmentsByAccount(accountID)
	if err != nil {
		return err
	}

	// For any Assignments with Status=ResetLock,
	// change the status back to "Active"
	for _, assgn := range assignments {
		// Only consider ResetLock'd assignments
		if assgn.AssignmentStatus != db.ResetLock {
			continue
		}

		// Set Status=ResetLock
		log.Printf("Setting Assignment Status from ResetLock to Active: %s - %s",
			accountID, assgn.UserID)
		_, err := dbSvc.TransitionAssignmentStatus(
			accountID, assgn.UserID,
			db.ResetLock, db.Active,
		)
		if err != nil {
			return err
		}
	}

	// If the Account.Status=NotReady, change it back to Status=Ready
	log.Printf("Setting Account Status from NotReady to Ready: %s", accountID)
	_, err = dbSvc.TransitionAccountStatus(
		accountID,
		db.NotReady, db.Ready,
	)
	// Ignore StatusTransitionErrors
	// (just means the status was NOT previously NotReady")
	if _, ok := err.(*db.StatusTransitionError); !ok {
		return err
	}

	return nil
}

func nukeAccount(svc *service, isDryRun bool) error {
	// Generate the configuration of the yaml file using the template file
	// provided and substituting necessary phrases.
	configFile, err := generateNukeConfig(svc)
	if err != nil {
		return err
	}

	// Construct Nuke
	nuke := reset.Nuke{}

	// Configure the NukeAccountInput
	config := svc.config()
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

func generateNukeConfig(svc *service) (string, error) {
	config := svc.config()

	// Verify the nuke template configuration to download file from s3 or to
	// use the default
	var template string
	if config.nukeTemplateBucket != "STUB" && config.nukeTemplateKey != "STUB" {
		log.Printf("Using Nuke Configuration from S3: %s/%s",
			config.nukeTemplateBucket, config.nukeTemplateKey)

		// Download the file from S3
		template = fmt.Sprintf("/tmp/nuke-config-template-%s.yml",
			config.accountID)
		err := svc._s3Service().Download(config.nukeTemplateBucket,
			config.nukeTemplateKey, template)
		if err != nil {
			return "", err
		}
	} else {
		log.Printf("Using Default Nuke Configuration: %s",
			config.nukeTemplateDefault)

		// Use default template
		template = config.nukeTemplateDefault
	}

	// Generate the configuration of the yaml file using the template file
	// provided and substituting necessary phrases.
	subs := map[string]string{
		"{{id}}":         config.accountID,
		"{{admin_role}}": config.accountAdminRoleName,
		"{{user_role}}":  config.accountUserRoleName,
	}
	modConfig, err := reset.GenerateConfig(template, subs)
	if err != nil {
		log.Fatalf("%s  :  %s\n", config.accountID, err)
	}
	log.Println(string(modConfig))
	configFile := fmt.Sprintf("/tmp/nuke-config-%s.yml", config.accountID)
	err = ioutil.WriteFile(configFile, modConfig, 0666)
	if err != nil {
		return "", err
	}
	return configFile, nil
}
