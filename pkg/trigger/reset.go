package trigger

import (
	"errors"
	"log"

	"github.com/Optum/Redbox/pkg/db"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/Optum/Redbox/pkg/common"
)

// ResetInput is the container used for the Queue and Pipeline implementations
// to execute a Reset for an AWS Account
type ResetInput struct {
	ResetQueue    common.Queue
	ResetQueueURL *string
	ResetBuild    common.Builder
	BuildName     *string
	DbSvc         db.DBer
}

// ResetResult is the individual results of a Reset trigger for an AWS
// Account, including if the CodeBuild build has started executing and
// if the SQS Message was deleted from the Queue.
type ResetResult struct {
	BuildTrigger    bool
	MessageDeletion bool
}

// ResetOutput is the overall results of the Reset function containing the
// overall Success and a list of Accounts their respective results.
type ResetOutput struct {
	Success  bool
	Accounts map[string]ResetResult
}

// Reset will drain Messages from the provided Queue and execute the respective
// Pipeline for each Message. Each Message should be structured to contain the
// 12-Digit AWS Account ID in the Body.
func Reset(input *ResetInput) (*ResetOutput, error) {
	// Construct the ResetOutput to be returned with the Results of the whole
	// reset function
	output := ResetOutput{
		Success:  true,
		Accounts: make(map[string]ResetResult),
	}

	// Construct the messages to be received
	maxMessages := int64(10)
	messageInput := sqs.ReceiveMessageInput{
		QueueUrl:            input.ResetQueueURL,
		MaxNumberOfMessages: &maxMessages,
	}

	// Retrieve at most 10 messages from the Queue
	messages, err := input.ResetQueue.ReceiveMessage(&messageInput)
	if err != nil {
		output.Success = false
		return &output, err
	}

	// Do while messages can still be received
	for len(messages.Messages) > 0 {
		// Iterate through each Message and trigger their respective Code Pipeline
		// If there's an error, log it and update the status for the individual
		// result and move on to the next account.
		for _, message := range messages.Messages {
			// Assumes the Message's Body to just contain the Account's ID to Reset
			accountID := *message.Body
			result := ResetResult{
				BuildTrigger:    false,
				MessageDeletion: false,
			}
			log.Printf("Start Account: %s\nMessage ID: %s\n", accountID,
				*message.MessageId)

			// Mark any account assignments as Status=ResetLock
			err := resetLockAccount(input.DbSvc, accountID)
			if err != nil {
				log.Printf("Error: %s", err)
				output.Success = false
				output.Accounts[accountID] = result
				continue
			}

			// Construct Reset Build Inputs
			resetBuildEnvironment := map[string]string{
				"RESET_ACCOUNT": accountID,
			}

			// Trigger Code Pipeline
			log.Printf("Triggering Reset Build %s for Account %s\n",
				*input.BuildName, accountID)
			buildID, err := input.ResetBuild.StartBuild(input.BuildName,
				resetBuildEnvironment)
			if err != nil {
				log.Printf("Error: %s", err)
				output.Success = false
				output.Accounts[accountID] = result
				continue
			}
			result.BuildTrigger = true
			log.Printf("Triggered Build ID: %s\n", buildID)

			// Construct the message to be deleted
			deleteMessageInput := sqs.DeleteMessageInput{
				QueueUrl:      input.ResetQueueURL,
				ReceiptHandle: message.ReceiptHandle,
			}

			// Delete the Message
			_, err = input.ResetQueue.DeleteMessage(&deleteMessageInput)
			if err != nil {
				log.Printf("Error: %s", err)
				output.Success = false
				output.Accounts[accountID] = result
				continue
			}
			result.MessageDeletion = true
			log.Printf("Deleted Message: %s\n", *message.MessageId)
			log.Printf("End Account: %s\n", accountID)

			// Add the account to the output
			output.Accounts[accountID] = result
		}

		// Retrieve at most 10 messages from the Queue
		messages, err = input.ResetQueue.ReceiveMessage(&messageInput)
		if err != nil {
			output.Success = false
			return &output, err
		}

	}

	// Return an error of the overall success was not true
	if !output.Success {
		return &output, errors.New("Error: Could not successfully trigger a " +
			"reset on all accounts")
	}
	return &output, nil
}

// resetLockAccount will Reset Lock an Active/FinanceLock assignment to
// ResetLock/ResetFinanceLock status
func resetLockAccount(dbSvc db.DBer, accountID string) error {
	// Find all assignments for this account
	assignments, err := dbSvc.FindAssignmentsByAccount(accountID)
	if err != nil {
		return err
	}

	// Find Active and FinanceLock assignments
	for _, assgn := range assignments {
		// If Active, set status to ResetLock
		// Else if FinanceLock, set status to ResetFinanceLock
		if assgn.AssignmentStatus == db.Active {
			_, err := dbSvc.TransitionAssignmentStatus(
				accountID, assgn.UserID,
				db.Active, db.ResetLock,
			)
			if err != nil {
				return err
			}
		} else if assgn.AssignmentStatus == db.FinanceLock {
			_, err := dbSvc.TransitionAssignmentStatus(
				accountID, assgn.UserID,
				db.FinanceLock, db.ResetFinanceLock,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
