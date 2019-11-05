package main

import (
	"log"

	"github.com/pkg/errors"

	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// addAccountToQueue publishes a single account ID into the SQS
// as an event for consumption
func addAccountToQueue(accounts []*db.Account, queueURL *string,
	queue common.Queue, dbSvc db.DBer) error {
	// For each Account, send the message to Reset Queue and update
	// FinanceLock Lease status if necessary
	for _, acct := range accounts {
		// Send Message
		err := queue.SendMessage(queueURL, &acct.ID)
		if err != nil {
			return errors.Wrapf(err, "Failed to add account %s to queue accounts", acct.ID)
		}
		log.Printf("%s : Added to Reset Queue\n", acct.ID)
	}
	return nil

}

// rbenqHandler is the base handler function for the lambda
func rbenqHandler(cloudWatchEvent events.CloudWatchEvent) error {

	// Create Database Service
	dbSvc, err := db.NewFromEnv()
	if err != nil {
		return err
	}

	// Get NotReady Accounts
	accounts, err := dbSvc.FindAccountsByStatus(db.NotReady)
	if err != nil {
		log.Printf("Failed to list accounts: %s\n", err)
		return err
	}

	// Create the Queue Service
	queueURL := common.RequireEnv("RESET_SQS_URL")
	awsSession := session.New()
	sqsClient := sqs.New(awsSession)
	queue := common.SQSQueue{
		Client: sqsClient,
	}

	// Enqueue accounts to be reset
	err = addAccountToQueue(accounts, &queueURL, queue, dbSvc)
	if err != nil {
		log.Printf("Failed to enqueue accounts: %s\n", err)
		return err
	}

	return nil
}

// Main
func main() {
	lambda.Start(rbenqHandler)
}
