package reset

import (
	"fmt"
	"log"
	"time"

	"github.com/avast/retry-go"
)

// LaunchpadAccountInput is the input type that is needed to run the
// LaunchpadAccount function and which account to apply to it.
type LaunchpadAccountInput struct {
	Launchpad     Launchpader
	AccountID     string
	MasterAccount string
	WaitSeconds   time.Duration
}

// LaunchpadAccount will trigger Launchpad to the provided account and will
// monitor the state of the account until completed
func LaunchpadAccount(input *LaunchpadAccountInput) error {
	// Setup Launchpad
	launchpad := input.Launchpad
	account := input.AccountID

	// Authenticate to use the LaunchpadService
	log.Println("Authenticate to Launchpad Service")
	bearer, err := launchpad.Authenticate()
	if err != nil {
		return fmt.Errorf("Error : Failed to authenticate - %s", err)
	}
	log.Println("Authenticate to Launchpad Service Complete")

	// Trigger Launchpad on the provided Account
	log.Printf("Trigger Launchpad for %s\n", account)
	var deployID string
	var errTrigger, errAuth error
	err = retry.Do(
		func() error {
			deployID, errTrigger = launchpad.TriggerLaunchpad(account,
				input.MasterAccount, bearer)
			if errTrigger != nil {
				log.Printf("Warning : Couldn't Deploy Launchpad to %s - %s",
					account, errTrigger)

				// Reauthenticate
				bearer, errAuth = launchpad.Authenticate()
				if errAuth != nil {
					log.Printf("Warning : Failed to Authenticate - %s\n",
						errAuth)
					return errAuth
				}
				return errTrigger
			}
			return nil
		},
		retry.Attempts(3), // Retry 3 times
		retry.Delay(input.WaitSeconds*time.Second), // Retry every ? seconds
		retry.LastErrorOnly(true),                  // Only return the last error
	)
	if err != nil {
		return fmt.Errorf("Error : Couldn't Deploy Launchpad to %s - %s",
			account, err)
	}
	log.Printf("Trigger Launchpad for %s Complete: Build ID - %s\n", account,
		deployID)

	// Wait every 15 seconds to check the status of the Launchpad.
	// Will only exit loop if there's a success.
	loop := true
	var deployStatus string
	var errCheck error
	for loop {
		// This is a lenghty process and will be making multiple HTTP calls
		// and in order to circumvent random misses with calls, a retry will
		// be set in place.
		err = retry.Do(
			func() error {
				// Get the deploy status
				deployStatus, errCheck = launchpad.CheckLaunchpad(account,
					deployID, bearer)

				// If there's an error, reauthenticate and return last error
				if errCheck != nil {
					log.Printf("Warning : Failed to get Deployment %s - %s\n",
						deployID, errCheck)

					// Reauthenticate
					bearer, errAuth = launchpad.Authenticate()
					if errAuth != nil {
						log.Printf("Warning : Failed to Authenticate - %s\n",
							errAuth)
						return errAuth
					}
					return errCheck
				}
				return nil
			},
			retry.Attempts(5), // Retry 5 times
			retry.Delay(input.WaitSeconds*time.Second), // Retry every ? seconds
			retry.LastErrorOnly(true),                  // Only return the last error
		)
		if err != nil {
			return fmt.Errorf("Error : Failed Deploying Launchpad to %s - %s",
				account, err)
		}
		log.Printf("Launchpad Status Build %s : %s\n", deployID, deployStatus)

		// Exit loop if successful
		if deployStatus == "SUCCESS" {
			break
		}

		// Error if build is not in progress
		switch deployStatus {
		case "IN-PROGRESS": // Do nothing and continue
		case "ABORTED", "FAILURE", "UNSTABLE", "NOT_BUILT":
			return fmt.Errorf("Error : Failed Deploying Launchpad to %s"+
				" - %s Launchpad Build %s", account, deployStatus, deployID)
		default: // Unknown status - error out
			return fmt.Errorf("Error : Unknown Status Deploying Launchpad "+
				"to %s - %s Launchpad Build %s", account, deployStatus,
				deployID)
		}
		time.Sleep(input.WaitSeconds * time.Second)
	}
	log.Printf("Launchpad Success for %s - Build %s", account, deployID)

	return nil
}
