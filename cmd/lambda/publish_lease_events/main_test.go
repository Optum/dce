package main

import (
	"errors"
	"log"
	"testing"

	commonMocks "github.com/Optum/dce/pkg/common/mocks"
	"github.com/Optum/dce/pkg/db"
	dbMocks "github.com/Optum/dce/pkg/db/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const UnlockedSnsTopic string = "arn:aws:sns:us-east-1:123456789012:lease-unlocked"
const LockedSnsTopic string = "arn:aws:sns:us-east-1:123456789012:lease-locked"

func TestLeaseFromImageSuccess(t *testing.T) {

	// Arrange

	expectedOutput := db.Lease{
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

	var activeImage = map[string]events.DynamoDBAttributeValue{
		"AccountId":      events.NewStringAttribute("123456789012"),
		"principalId":    events.NewStringAttribute("TestPrincipalID"),
		"LeaseStatus":    events.NewStringAttribute("Active"),
		"createdOn":      events.NewNumberAttribute("1565723448"),
		"lastModifiedOn": events.NewNumberAttribute("1565723448"),
		"budgetAmount":   events.NewNumberAttribute("300.000"),
		"budgetCurrency": events.NewStringAttribute("USD"),
		// "budgetNotificationEmails": events.NewListAttribute(budgetNotificationEmails),
		"leaseStatusModifiedOn": events.NewNumberAttribute("1565723448"),
	}

	var inactiveImage = map[string]events.DynamoDBAttributeValue{
		"AccountId":      events.NewStringAttribute("123456789012"),
		"principalId":    events.NewStringAttribute("TestPrincipalID"),
		"LeaseStatus":    events.NewStringAttribute("Inactive"),
		"createdOn":      events.NewNumberAttribute("1565723448"),
		"lastModifiedOn": events.NewNumberAttribute("1565723448"),
		"budgetAmount":   events.NewNumberAttribute("300.000"),
		"budgetCurrency": events.NewStringAttribute("USD"),
		// "budgetNotificationEmails": events.NewListAttribute(budgetNotificationEmails),
		"leaseStatusModifiedOn": events.NewNumberAttribute("1565723448"),
	}

	activeToInactiveRecord := events.DynamoDBStreamRecord{
		OldImage: activeImage,
		NewImage: inactiveImage,
	}

	inactiveToActiveRecord := events.DynamoDBStreamRecord{
		OldImage: inactiveImage,
		NewImage: activeImage,
	}

	activeToInactiveEventRecord := events.DynamoDBEventRecord{
		EventName: "MODIFY",
		Change:    activeToInactiveRecord,
	}

	inactiveToActiveEventRecord := events.DynamoDBEventRecord{
		EventName: "MODIFY",
		Change:    inactiveToActiveRecord,
	}

	sqsSvc := &commonMocks.Queue{}
	snsSvc := &commonMocks.Notificationer{}
	dbSvc := &dbMocks.DBer{}

	tests := []struct {
		name                 string
		args                 args
		wantErr              bool
		shoudEnqueueReset    bool
		shouldErrorOnEnqueue bool
		expectedSnsTopic     string
		transitionAccount    *db.Account
	}{
		{
			name: "Happy path...",
			args: args{
				&handleRecordInput{
					record:                activeToInactiveEventRecord,
					snsSvc:                snsSvc,
					sqsSvc:                sqsSvc,
					dbSvc:                 dbSvc,
					leaseLockedTopicArn:   LockedSnsTopic,
					leaseUnlockedTopicArn: UnlockedSnsTopic,
					resetQueueURL:         "sqs-queue",
				},
			},
			wantErr:           false,
			shoudEnqueueReset: true,
			expectedSnsTopic:  LockedSnsTopic,
			transitionAccount: &db.Account{
				ID: "123456789012",
			},
		},
		{
			name: "Went from inactive to active...",
			args: args{
				&handleRecordInput{
					record:                inactiveToActiveEventRecord,
					snsSvc:                snsSvc,
					sqsSvc:                sqsSvc,
					dbSvc:                 dbSvc,
					leaseLockedTopicArn:   LockedSnsTopic,
					leaseUnlockedTopicArn: UnlockedSnsTopic,
					resetQueueURL:         "sqs-queue",
				},
			},
			wantErr:           false,
			shoudEnqueueReset: false,
			expectedSnsTopic:  UnlockedSnsTopic,
			transitionAccount: &db.Account{
				ID: "123456789012",
			},
		},
		{
			name: "Throwing error on enqueue",
			args: args{
				&handleRecordInput{
					record:                activeToInactiveEventRecord,
					snsSvc:                snsSvc,
					sqsSvc:                sqsSvc,
					dbSvc:                 dbSvc,
					leaseLockedTopicArn:   LockedSnsTopic,
					leaseUnlockedTopicArn: UnlockedSnsTopic,
					resetQueueURL:         "sqs-err-queue",
				},
			},
			wantErr:              true,
			shoudEnqueueReset:    true,
			shouldErrorOnEnqueue: true,
			expectedSnsTopic:     LockedSnsTopic,
			transitionAccount: &db.Account{
				ID: "123456789012",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.shoudEnqueueReset {
				// If it should enqueue the reset, then it should also flip the account status
				// to Inactive.
				dbSvc.On("TransitionAccountStatus",
					"123456789012",
					db.Leased,
					db.NotReady,
				).Return(tt.transitionAccount, nil)

				if tt.shouldErrorOnEnqueue {
					sqsSvc.On("SendMessage", aws.String(tt.args.input.resetQueueURL), aws.String("{\"Id\":\"123456789012\",\"AccountStatus\":\"\",\"LastModifiedOn\":0,\"CreatedOn\":0,\"AdminRoleArn\":\"\",\"PrincipalRoleArn\":\"\",\"PrincipalPolicyHash\":\"\",\"Metadata\":null}")).Return(errors.New("error enqueuing message"))
				} else {
					sqsSvc.On("SendMessage", aws.String(tt.args.input.resetQueueURL), aws.String("{\"Id\":\"123456789012\",\"AccountStatus\":\"\",\"LastModifiedOn\":0,\"CreatedOn\":0,\"AdminRoleArn\":\"\",\"PrincipalRoleArn\":\"\",\"PrincipalPolicyHash\":\"\",\"Metadata\":null}")).Return(errors.New("error enqueuing message")).Return(nil)
				}
			}
			snsSvc.On("PublishMessage", &tt.expectedSnsTopic, mock.Anything, true).Return(nil, nil)

			err := handleRecord(tt.args.input)
			log.Printf("Got err value from handleRecord: %s", err)
			sqsSvc.AssertExpectations(t)
			snsSvc.AssertExpectations(t)
			dbSvc.AssertExpectations(t)

			if (err != nil) != tt.wantErr {
				t.Errorf("handleRecord() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
