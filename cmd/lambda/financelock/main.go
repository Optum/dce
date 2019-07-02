package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
)

// Main Handler for the lambda
func finLockHandler(ctx context.Context, sqsEvt events.SQSEvent) error {
	var acct string
	// Determine the AWS acct #
	for _, message := range sqsEvt.Records {
		var raw map[string]interface{}
		err := json.Unmarshal([]byte(message.Body), &raw)
		if err != nil {
			log.Printf("JSON marshal of Account failed: %s", err)
			return err
		}
		acct = raw["Message"].(string)
	}

	// SQS Client
	var qSvc *common.SQSQueue
	err := qSvc.NewFromEnv()
	if err != nil {
		log.Printf("SQS client creation failed: %s", err)
		return err
	}

	// Create the Database Service from the environment
	dbSvc, errDB := db.NewFromEnv()
	if errDB != nil {
		log.Printf("Database init failed: %s", errDB)
		return errDB
	}

	// Update Database
	err = updateAssignment(acct, dbSvc)
	if err != nil {
		log.Printf("Database Assignment failed: %s", err)
		return err
	}

	// URL to our queue
	qURL := common.RequireEnv("RESET_SQS_URL")

	// Push to SQS
	err = qSvc.SendMessage(&qURL, aws.String(acct))
	if err != nil {
		log.Printf("Error: %s", err)
		return err
	}

	return nil
}

// Check for active assignments for user and update to FinanceLock status in database
func updateAssignment(acct string, dbSvc db.DBer) error {
	checkAssignment, err := dbSvc.FindAssignmentsByAccount(acct)
	if err != nil {
		log.Printf("Failed to determine User Assignment to Account %s", acct)
		return err
	}

	locked := false
	for _, r := range checkAssignment {
		if r.AssignmentStatus == db.Active {
			// Set the Account as FinanceLock'd
			log.Printf("Set Assignment %s Status to FinanceLock for User %s\n", r.UserID,
				r.AccountID)
			_, err = dbSvc.TransitionAssignmentStatus(r.AccountID, r.UserID, db.Active, db.FinanceLock)
			if err != nil {
				log.Printf("Failed Finance Lock for user %s on account %s",
					r.UserID, r.AccountID)
				return err
			}
			locked = true
		}
	}
	if locked == false {
		log.Printf("No active assignment to finance lock for account %s", checkAssignment[0].AccountID)
		return fmt.Errorf("No active assignment to finance lock for account %s", checkAssignment[0].AccountID)
	}
	// Success case
	return nil
}

// Main
func main() {
	lambda.Start(finLockHandler)
}
