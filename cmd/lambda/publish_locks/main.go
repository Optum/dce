package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/Optum/Dcs/pkg/common"
	"github.com/Optum/Dcs/pkg/db"
	errors2 "github.com/Optum/Dcs/pkg/errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/sns"
)

// Start the Lambda Handler
func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, event events.DynamoDBEvent) error {
	// Defer errors for later
	deferredErrors := []error{}

	awsSession := session.Must(session.NewSession())
	leaseLockedTopicArn := common.RequireEnv("LEASE_LOCKED_TOPIC_ARN")
	leaseUnlockedTopicArn := common.RequireEnv("LEASE_UNLOCKED_TOPIC_ARN")

	// We get a stream of DynDB records, representing changes to the table
	for _, record := range event.Records {

		input := handleRecordInput{
			record:                record,
			leaseLockedTopicArn:   leaseLockedTopicArn,
			leaseUnlockedTopicArn: leaseUnlockedTopicArn,
			snsSvc:                &common.SNS{Client: sns.New(awsSession)},
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
	leaseLockedTopicArn   string
	leaseUnlockedTopicArn string
}

func handleRecord(input *handleRecordInput) error {
	record := input.record
	dcsLease, err := leaseFromImage(record.Change.NewImage)
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

		// Lease was locked if it transitioned from "Active" --> "*Lock"
		wasLocked := isActiveStatus(prevLeaseStatus) && isLockStatus(nextLeaseStatus)
		// Lease was unlocked if it transitioned from "*Lock" --> "Active"
		wasUnlocked := isLockStatus(prevLeaseStatus) && isActiveStatus(nextLeaseStatus)

		publishInput := publishLeaseInput{
			lease:  dcsLease,
			snsSvc: input.snsSvc,
		}
		if wasLocked {
			publishInput.topicArn = input.leaseLockedTopicArn
			err := publishLease(&publishInput)
			if err != nil {
				return err
			}
		}

		if wasUnlocked {
			publishInput.topicArn = input.leaseUnlockedTopicArn
			err := publishLease(&publishInput)
			if err != nil {
				return err
			}
		}
	default:
	}

	return nil
}

func leaseFromImage(image map[string]events.DynamoDBAttributeValue) (*db.DcsLease, error) {

	dcsLease, err := UnmarshalStreamImage(image)
	if err != nil {
		return nil, err
	}

	return dcsLease, nil

}

func isLockStatus(status string) bool {
	switch status {
	case string(db.ResetLock),
		string(db.FinanceLock),
		string(db.ResetFinanceLock):
		return true
	}

	return false
}

func isActiveStatus(status string) bool {
	return status == string(db.Active)
}

type publishLeaseInput struct {
	snsSvc   common.Notificationer
	topicArn string
	lease    *db.DcsLease
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
func UnmarshalStreamImage(attribute map[string]events.DynamoDBAttributeValue) (*db.DcsLease, error) {

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

	out := db.DcsLease{}
	dynamodbattribute.UnmarshalMap(dbAttrMap, &out)
	return &out, nil

}
