package main

import (
	"testing"

	commonMocks "github.com/Optum/Redbox/pkg/common/mocks"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLeaseFromImageSuccess(t *testing.T) {

	// Arrange

	expectedOutput := db.RedboxLease{
		AccountID:                "TestAccountID",
		PrincipalID:              "TestPrincipalID",
		LeaseStatus:              db.Inactive,
		CreatedOn:                1565723448,
		LastModifiedOn:           1565723448,
		BudgetAmount:             300,
		BudgetCurrency:           "USD",
		BudgetNotificationEmails: []string{"recipA@example.com", "recipB@example.com"},
		LeaseStatusModifiedOn:    1565723448,
	}

	email1 := events.NewStringAttribute("recipA@example.com")
	email2 := events.NewStringAttribute("recipB@example.com")
	budgetNotificationEmails := []events.DynamoDBAttributeValue{email1, email2}

	var input = map[string]events.DynamoDBAttributeValue{
		"accountId":                events.NewStringAttribute("TestAccountID"),
		"principalId":              events.NewStringAttribute("TestPrincipalID"),
		"LeaseStatus":              events.NewStringAttribute("Inactive"),
		"createdOn":                events.NewNumberAttribute("1565723448"),
		"lastModifiedOn":           events.NewNumberAttribute("1565723448"),
		"budgetAmount":             events.NewNumberAttribute("300.000"),
		"budgetCurrency":           events.NewStringAttribute("USD"),
		"budgetNotificationEmails": events.NewListAttribute(budgetNotificationEmails),
		"leaseStatusModifiedOn":    events.NewNumberAttribute("1565723448"),
	}

	actualOutput, err := leaseFromImage(input)

	assert.Nil(t, err)
	assert.Equal(t, actualOutput, &expectedOutput)
}

func Test_handleRecord(t *testing.T) {
	type args struct {
		input *handleRecordInput
	}

	var oldImage = map[string]events.DynamoDBAttributeValue{
		"AccountId":      events.NewStringAttribute("TestAccountID"),
		"principalId":    events.NewStringAttribute("TestPrincipalID"),
		"LeaseStatus":    events.NewStringAttribute("Active"),
		"createdOn":      events.NewNumberAttribute("1565723448"),
		"lastModifiedOn": events.NewNumberAttribute("1565723448"),
		"budgetAmount":   events.NewNumberAttribute("300.000"),
		"budgetCurrency": events.NewStringAttribute("USD"),
		// "budgetNotificationEmails": events.NewListAttribute(budgetNotificationEmails),
		"leaseStatusModifiedOn": events.NewNumberAttribute("1565723448"),
	}

	var newImage = map[string]events.DynamoDBAttributeValue{
		"AccountId":      events.NewStringAttribute("TestAccountID"),
		"principalId":    events.NewStringAttribute("TestPrincipalID"),
		"LeaseStatus":    events.NewStringAttribute("Inactive"),
		"createdOn":      events.NewNumberAttribute("1565723448"),
		"lastModifiedOn": events.NewNumberAttribute("1565723448"),
		"budgetAmount":   events.NewNumberAttribute("300.000"),
		"budgetCurrency": events.NewStringAttribute("USD"),
		// "budgetNotificationEmails": events.NewListAttribute(budgetNotificationEmails),
		"leaseStatusModifiedOn": events.NewNumberAttribute("1565723448"),
	}

	record := events.DynamoDBStreamRecord{
		OldImage: oldImage,
		NewImage: newImage,
	}

	sampleRecord := events.DynamoDBEventRecord{
		EventName: "MODIFY",
		Change:    record,
	}

	sqsSvc := &commonMocks.Queue{}
	snsSvc := &commonMocks.Notificationer{}

	input := &handleRecordInput{
		record:                sampleRecord,
		snsSvc:                snsSvc,
		sqsSvc:                sqsSvc,
		leaseLockedTopicArn:   "arn:aws:sns:us-east-1:123456789012:lease-locked",
		leaseUnlockedTopicArn: "arn:aws:sns:us-east-1:123456789012:lease-unlocked",
		resetQueueURL:         "sqs-queue",
	}
	happyPathArgs := &args{
		input: input,
	}

	tests := []struct {
		name              string
		args              args
		wantErr           bool
		shoudEnqueueReset bool
	}{
		{"Happy path...", *happyPathArgs, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mocks
			snsSvc.On("PublishMessage", &input.leaseLockedTopicArn, mock.Anything, true).Return(nil, nil)

			if tt.shoudEnqueueReset {
				sqsSvc.On("SendMessage", aws.String(input.resetQueueURL), mock.Anything).Return(nil)
			}

			err := handleRecord(tt.args.input)
			sqsSvc.AssertExpectations(t)
			snsSvc.AssertExpectations(t)

			if (err != nil) != tt.wantErr {
				t.Errorf("handleRecord() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
