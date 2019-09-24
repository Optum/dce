package main

import (
	"log"

	"github.com/pkg/errors"

	"github.com/Optum/Dcs/pkg/common"
	"github.com/Optum/Dcs/pkg/db"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// enqueueDcses publishes a single dcs struct into the SQS
// as an event for consumption
func enqueueDcses(dcses []*db.DcsAccount, queueURL *string,
	queue common.Queue, dbSvc db.DBer) error {
	// For each Dcs Account, send the message to Reset Queue and update
	// FinanceLock Lease status if necessary
	for _, dcs := range dcses {
		// Send Message
		err := queue.SendMessage(queueURL, &dcs.ID)
		if err != nil {
			return errors.Wrap(err, "Failed to enqueue accounts")
		}
		log.Printf("%s : Added to Reset Queue\n", dcs.ID)

		// Transition FinanceLock Lease if needed
		log.Printf("%s : Checking for Finance Lock\n", dcs.ID)
		err = transitionFinanceLock(dcs.ID, dbSvc)
		if err != nil {
			return errors.Wrap(err, "Failed to enqueue accounts")
		}
	}
	return nil

}

// transitionFinanceLock is a helper function to that will transition a
// FinanceLock Account Lease to Active if one exists
func transitionFinanceLock(accountID string, dbSvc db.DBer) error {
	// Find all leases
	leases, err := dbSvc.FindLeasesByAccount(accountID)
	if err != nil {
		return err
	}

	// Look for a FinanceLock Lease and transition its state to Active
	for _, lease := range leases {
		if lease.LeaseStatus == db.FinanceLock || lease.LeaseStatus == db.ResetFinanceLock {
			_, err = dbSvc.TransitionLeaseStatus(accountID,
				lease.PrincipalID, lease.LeaseStatus, db.Active)
			if err != nil {
				return err
			}
			log.Printf("%s : Removed Finance Lock\n", accountID)
			return nil
		}
	}
	log.Printf("%s : No Finance Lock\n", accountID)
	return nil
}

// rbenqHandler is the base handler function for the lambda
func rbenqHandler(cloudWatchEvent events.CloudWatchEvent) error {

	// Create Database Service
	dbSvc, err := db.NewFromEnv()
	if err != nil {
		return err
	}

	// Get Dcses
	dcses, err := dbSvc.GetAccountsForReset()
	if err != nil {
		log.Printf("Failed to get Dcses: %s\n", err)
		return err
	}

	// Create the Queue Service
	queueURL := common.RequireEnv("RESET_SQS_URL")
	awsSession := session.New()
	sqsClient := sqs.New(awsSession)
	queue := common.SQSQueue{
		Client: sqsClient,
	}

	// Enqueue dcses to be reset
	err = enqueueDcses(dcses, &queueURL, queue, dbSvc)
	if err != nil {
		log.Printf("Failed to enqueue dcses: %s\n", err)
		return err
	}

	return nil
}

// Main
func main() {
	lambda.Start(rbenqHandler)
}
