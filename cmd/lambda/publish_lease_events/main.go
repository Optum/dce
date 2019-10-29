package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"
	errors2 "github.com/Optum/Redbox/pkg/errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// Start the Lambda Handler
func main() {
	lambda.Start(handler)
}

// handler Handles the update to the DynamoDBEvent, which in this
// case will send a message to an SQS queue if, and only if, the
// status of the lease has been flipped to Inactive (for cleanup)
// and then will route the message to the correct SNS topic.
func handler(ctx context.Context, event events.DynamoDBEvent) error {
	// Defer errors for later
	deferredErrors := []error{}

	awsSession := session.Must(session.NewSession())
	leaseLockedTopicArn := common.RequireEnv("LEASE_LOCKED_TOPIC_ARN")
	leaseUnlockedTopicArn := common.RequireEnv("LEASE_UNLOCKED_TOPIC_ARN")
	resetQueueURL := common.RequireEnv("RESET_QUEUE_URL")
	dbSvc, err := db.NewFromEnv()
	if err != nil {
		log.Fatalf("Failed to configure DB service %s", err)
	}

	// We get a stream of DynDB records, representing changes to the table
	for _, record := range event.Records {

		input := handleRecordInput{
			record:                record,
			leaseLockedTopicArn:   leaseLockedTopicArn,
			leaseUnlockedTopicArn: leaseUnlockedTopicArn,
			resetQueueURL:         resetQueueURL,
			snsSvc:                &common.SNS{Client: sns.New(awsSession)},
			sqsSvc:                &common.SQSQueue{Client: sqs.New(awsSession)},
			dbSvc:                 dbSvc,
		}
		err := handleRecord(&input)
		if err != nil {
			deferredErrors = append(deferredErrors, err)
		}
	}

	if len(deferredErrors) > 0 {
		return errors2.NewMultiError("Failed to handle DynDB Event", deferredErrors)
	}

	return nil
}

type handleRecordInput struct {
	record                events.DynamoDBEventRecord
	snsSvc                common.Notificationer
	sqsSvc                common.Queue
	dbSvc                 db.DBer
	leaseLockedTopicArn   string
	leaseUnlockedTopicArn string
	resetQueueURL         string
}

func handleRecord(input *handleRecordInput) error {
	record := input.record
	redboxLease, err := leaseFromImage(record.Change.NewImage)
	if err != nil {
		return err
	}
	switch record.EventName {
	// We only care about modified records
	case "MODIFY":
		// Grab the previous lease status
		prevLeaseStatusAttr, ok := record.Change.OldImage["LeaseStatus"]
		if !ok {
			return errors.New("prev lease missing LeaseStatus attr")
		}
		prevLeaseStatus := prevLeaseStatusAttr.String()

		// Grab the new lease status
		nextLeaseStatusAttr, ok := record.Change.NewImage["LeaseStatus"]
		if !ok {
			return errors.New("next lease missing LeaseStatus attr")
		}
		nextLeaseStatus := nextLeaseStatusAttr.String()

		if prevLeaseStatus == nextLeaseStatus {
			log.Print("Lease status has not changed.")
			return nil
		}

		log.Printf("Transitioning from %s to %s", prevLeaseStatus, nextLeaseStatus)

		// Lease is now expired if it transitioned from "Active" --> "Inactive"
		isExpired := isActiveStatus(prevLeaseStatus) && !isActiveStatus(nextLeaseStatus)

		if isExpired {
			// Before adding the account to any queues, make sure the account is
			// updated to NotReady state.
			_, err = input.dbSvc.TransitionAccountStatus(redboxLease.AccountID, db.Leased, db.NotReady)

			// Put the message on the SQS queue ONLY IF the status has gone
			// to Inactive.
			log.Printf("Adding account %s to the reset queue", redboxLease.AccountID)
			err := input.sqsSvc.SendMessage(
				aws.String(input.resetQueueURL),
				aws.String(redboxLease.AccountID),
			)

			if err != nil {
				errMsg := fmt.Sprintf("Failed to add account to reset queue for lease %s @ %s: %s", redboxLease.PrincipalID, redboxLease.AccountID, err)
				log.Println(errMsg)
				// throw the error. Because if we could not enqueue the lease reset, we want
				// the Lambda to error out so it can be re-tried per the retry policy of
				// the event source.
				return errors.New(errMsg)
			}
		}

		publishInput := publishLeaseInput{
			lease:  redboxLease,
			snsSvc: input.snsSvc,
		}

		// Route the lease event to the correct ARN, now for backwards compatibility.
		if isExpired {
			publishInput.topicArn = input.leaseLockedTopicArn
		} else {
			publishInput.topicArn = input.leaseUnlockedTopicArn
		}
		err := publishLease(&publishInput)
		if err != nil {
			return err
		}
	default:
	}

	return nil
}

func leaseFromImage(image map[string]events.DynamoDBAttributeValue) (*db.RedboxLease, error) {

	redboxLease, err := UnmarshalStreamImage(image)
	if err != nil {
		return nil, err
	}

	return redboxLease, nil

}

func isActiveStatus(status string) bool {
	return status == string(db.Active)
}

type publishLeaseInput struct {
	snsSvc   common.Notificationer
	topicArn string
	lease    *db.RedboxLease
}

func publishLease(input *publishLeaseInput) error {
	// Prepare the SNS message body
	leaseLockedMsg, err := common.PrepareSNSMessageJSON(input.lease)
	if err != nil {
		log.Printf("Failed to prepare SNS message for lease %s @ %s: %s",
			input.lease.PrincipalID, input.lease.AccountID, err)
		return err
	}

	_, err = input.snsSvc.PublishMessage(&input.topicArn, &leaseLockedMsg, true)
	if err != nil {
		log.Printf("Failed to publish SNS message for lease %s @ %s: %s",
			input.lease.PrincipalID, input.lease.AccountID, err)
		return err
	}
	return nil
}

// UnmarshalStreamImage converts events.DynamoDBAttributeValue to struct
func UnmarshalStreamImage(attribute map[string]events.DynamoDBAttributeValue) (*db.RedboxLease, error) {

	dbAttrMap := make(map[string]*dynamodb.AttributeValue)

	for k, v := range attribute {

		var dbAttr dynamodb.AttributeValue

		bytes, marshalErr := v.MarshalJSON()
		if marshalErr != nil {
			log.Printf("marshall error %v: %v", v, marshalErr)
			return nil, marshalErr
		}

		json.Unmarshal(bytes, &dbAttr)
		dbAttrMap[k] = &dbAttr
	}

	out := db.RedboxLease{}
	dynamodbattribute.UnmarshalMap(dbAttrMap, &out)
	return &out, nil

}
