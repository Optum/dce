package main

import (
	"log"

	"github.com/pkg/errors"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"
)

// enqueueRedboxes publishes a single redbox struct into the SQS
// as an event for consumption
func enqueueRedboxes(redboxes []*db.RedboxAccount, queueURL *string,
	queue common.Queue, dbSvc db.DBer) error {
	// For each Redbox Account, send the message to Reset Queue and update
	// FinanceLock Assignment status if necessary
	for _, redbox := range redboxes {
		// Send Message
		err := queue.SendMessage(queueURL, &redbox.ID)
		if err != nil {
			return errors.Wrap(err, "Failed to enqueue accounts")
		}
		log.Printf("%s : Added to Reset Queue\n", redbox.ID)

		// Transition FinanceLock Assignment if needed
		log.Printf("%s : Checking for Finance Lock\n", redbox.ID)
		err = transitionFinanceLock(redbox.ID, dbSvc)
		if err != nil {
			return errors.Wrap(err, "Failed to enqueue accounts")
		}
	}
	return nil

}

// transitionFinanceLock is a helper function to that will transition a
// FinanceLock Account Assignment to Active if one exists
func transitionFinanceLock(accountID string, dbSvc db.DBer) error {
	// Find all assignments
	assignments, err := dbSvc.FindAssignmentsByAccount(accountID)
	if err != nil {
		return err
	}

	// Look for a FinanceLock Assignment and transition its state to Active
	for _, assignment := range assignments {
		if assignment.AssignmentStatus == db.FinanceLock {
			_, err = dbSvc.TransitionAssignmentStatus(accountID,
				assignment.UserID, db.FinanceLock, db.Active)
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

	// Get Redboxes
	redboxes, err := dbSvc.GetAccountsForReset()
	if err != nil {
		log.Printf("Failed to get Redboxes: %s\n", err)
		return err
	}

	// Create the Queue Service
	queueURL := common.RequireEnv("RESET_SQS_URL")
	awsSession := session.New()
	sqsClient := sqs.New(awsSession)
	queue := common.SQSQueue{
		Client: sqsClient,
	}

	// Enqueue redboxes to be reset
	err = enqueueRedboxes(redboxes, &queueURL, queue, dbSvc)
	if err != nil {
		log.Printf("Failed to enqueue redboxes: %s\n", err)
		return err
	}

	return nil
}

// Main
func main() {
	lambda.Start(rbenqHandler)
}
