package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	errors2 "github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
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
	lease, err := leaseFromImage(record.Change.NewImage)
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
		didBecomeInactive := isActiveStatus(prevLeaseStatus) && !isActiveStatus(nextLeaseStatus)

		publishInput := publishLeaseInput{
			lease:  lease,
			snsSvc: input.snsSvc,
		}

		// Route the lease event to the correct ARN, now for backwards compatibility.
		if didBecomeInactive {
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


	log.Println("Handler complete. Let's see what's up with the account")
	acc2, err := input.dbSvc.GetAccount(lease.AccountID)
	log.Printf("%+v, %s", acc2, err)

	return nil
}

func leaseFromImage(image map[string]events.DynamoDBAttributeValue) (*db.Lease, error) {

	lease, err := UnmarshalStreamImage(image)
	if err != nil {
		return nil, err
	}

	return lease, nil

}

func isActiveStatus(status string) bool {
	return status == string(db.Active)
}

type publishLeaseInput struct {
	snsSvc   common.Notificationer
	topicArn string
	lease    *db.Lease
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
func UnmarshalStreamImage(attribute map[string]events.DynamoDBAttributeValue) (*db.Lease, error) {

	dbAttrMap := make(map[string]*dynamodb.AttributeValue)

	for k, v := range attribute {

		var dbAttr dynamodb.AttributeValue

		bytes, marshalErr := v.MarshalJSON()
		if marshalErr != nil {
			log.Printf("marshall error %v: %v", v, marshalErr)
			return nil, marshalErr
		}

		err := json.Unmarshal(bytes, &dbAttr)
		if err != nil {
			log.Printf("unmarshal error %v: %v", v, marshalErr)
			return nil, marshalErr
		}
		dbAttrMap[k] = &dbAttr
	}

	out := db.Lease{}
	err := dynamodbattribute.UnmarshalMap(dbAttrMap, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil

}
