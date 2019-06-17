// Package main is the Reset Pipeline used with CodeBuild under the Reset
// CodePipeline for an account
package main

import (
	"fmt"
	"github.com/avast/retry-go"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/reset"
	"io/ioutil"
	"log"
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
	// Note that
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
	config := svc.config()

	// Generate the configuration of the yaml file using the template file
	// provided and substituting necessary phrases.
	configFile, err := generateNukeConfig(config)
	if err != nil {
		return err
	}

	// Construct Nuke
	nuke := reset.Nuke{}

	// Configure the NukeAccountInput
	nukeAccountInput := reset.NukeAccountInput{
		AccountID:  config.accountID,
		RoleName:   config.customerRoleName,
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

func generateNukeConfig(config *serviceConfig) (string, error) {
	// Generate the configuration of the yaml file using the template file
	// provided and substituting necessary phrases.
	accountSubs := map[string]string{
		"{{id}}": config.accountID,
	}
	modConfig, err := reset.GenerateConfig(config.nukeTemplate, accountSubs)
	if err != nil {
		log.Fatalf("%s  :  %s\n", config.accountID, err)
	}
	log.Println(string(modConfig))
	configFile := fmt.Sprintf("/tmp/redbox-config-%s.yml", config.accountID)
	err = ioutil.WriteFile(configFile, modConfig, 0666)
	if err != nil {
		return "", err
	}
	return configFile, nil
}
