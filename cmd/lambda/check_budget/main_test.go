package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	awsMocks "github.com/Optum/Dce/pkg/awsiface/mocks"
	budgetMocks "github.com/Optum/Dce/pkg/budget/mocks"
	"github.com/Optum/Dce/pkg/common"
	commonMocks "github.com/Optum/Dce/pkg/common/mocks"
	"github.com/Optum/Dce/pkg/db"
	dbMocks "github.com/Optum/Dce/pkg/db/mocks"
	"github.com/Optum/Dce/pkg/email"
	emailMocks "github.com/Optum/Dce/pkg/email/mocks"
	"github.com/Optum/Dce/pkg/usage"
	usageMocks "github.com/Optum/Dce/pkg/usage/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCheckBudget(t *testing.T) {

	// Configure email templates
	emailTemplateHTML := `
<p>
{{if .IsOverBudget}}
AWS Dce Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded its budget of ${{.Lease.BudgetAmount}}. Actual spend is ${{.ActualSpend}}
{{else}}
AWS Dce Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded the {{.ThresholdPercentile}}% threshold limit for its budget of ${{.Lease.BudgetAmount}}.
Actual spend is ${{.ActualSpend}}
{{end}}
</p>
`
	emailTemplateText := `
{{if .IsOverBudget}}
AWS Dce Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded its budget of ${{.Lease.BudgetAmount}}. Actual spend is ${{.ActualSpend}}
{{else}}
AWS Dce Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded the {{.ThresholdPercentile}}% threshold limit for its budget of ${{.Lease.BudgetAmount}}.
Actual spend is ${{.ActualSpend}}
{{end}}
`
	emailTemplateSubject := `
AWS Dce Lease {{if .IsOverBudget}}over budget{{else}}at {{.ThresholdPercentile}}% of budget{{end}} [{{.Lease.AccountID}}]
`

	expectedOverBudgetEmailHTML := strings.TrimSpace(`
<p>

AWS Dce Lease for principal test-user in AWS Account 1234567890
has exceeded its budget of $100. Actual spend is $150

</p>
`)
	expectedOverBudgetEmailText := strings.TrimSpace(`
AWS Dce Lease for principal test-user in AWS Account 1234567890
has exceeded its budget of $100. Actual spend is $150
`)
	expectedOverBudgetText := "AWS Dce Lease over budget [1234567890]"

	type checkBudgetTestInput struct {
		budgetAmount                  float64
		actualSpend                   float64
		leaseStatus                   db.LeaseStatus
		expectedLeaseStatusTransition db.LeaseStatus
		shouldTransitionLeaseStatus   bool
		transitionLeaseError          error
		shouldSNS                     bool
		snsError                      error
		shouldSQSReset                bool
		sqsError                      error
		shouldSendEmail               bool
		expectedEmailSubject          string
		expectedEmailBodyHTML         string
		expectedEmailBodyText         string
		expectedError                 string
	}

	checkBudgetTest := func(test *checkBudgetTestInput) {
		dbSvc := &dbMocks.DBer{}
		tokenSvc := &commonMocks.TokenService{}
		budgetSvc := &budgetMocks.Service{}
		usageSvc := &usageMocks.Service{}
		snsSvc := &commonMocks.Notificationer{}
		sqsSvc := &awsMocks.SQSAPI{}
		emailSvc := &emailMocks.Service{}
		input := &lambdaHandlerInput{
			dbSvc: dbSvc,
			lease: &db.DceLease{
				AccountID:                "1234567890",
				PrincipalID:              "test-user",
				LeaseStatus:              test.leaseStatus,
				BudgetAmount:             test.budgetAmount,
				BudgetCurrency:           "USD",
				BudgetNotificationEmails: []string{"recipA@example.com", "recipB@example.com"},
				LeaseStatusModifiedOn:    time.Unix(100, 0).Unix(),
			},
			awsSession:                             &awsMocks.AwsSession{},
			tokenSvc:                               tokenSvc,
			budgetSvc:                              budgetSvc,
			usageSvc:                               usageSvc,
			snsSvc:                                 snsSvc,
			leaseLockedTopicArn:                    "lease-locked",
			sqsSvc:                                 sqsSvc,
			resetQueueURL:                          "reset-queue-url",
			emailSvc:                               emailSvc,
			budgetNotificationFromEmail:            "from@example.com",
			budgetNotificationBCCEmails:            []string{"bcc@example.com"},
			budgetNotificationTemplateHTML:         emailTemplateHTML,
			budgetNotificationTemplateText:         emailTemplateText,
			budgetNotificationTemplateSubject:      emailTemplateSubject,
			budgetNotificationThresholdPercentiles: []float64{75, 100},
		}

		// Should grab the account from the DB, to get it's adminRoleArn
		dbSvc.On("GetAccount", "1234567890").
			Return(&db.DceAccount{
				AdminRoleArn: "mock:admin:role:arn",
			}, nil)

		// Mock the TokenService
		// Should assume Account.AdminRoleArn
		tokenSvc.MockNewSession("mock:admin:role:arn")

		// Mock the BudgetService, actualSpend=150 (over budget)
		// Should use assumed role
		budgetSvc.On("SetCostExplorer", mock.Anything)
		currentTime := time.Now()
		startDate := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC)
		usageEndDate := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 23, 59, 59, 0, time.UTC)
		endDate := startDate.AddDate(0, 0, 1)
		budgetSvc.On("CalculateTotalSpend",
			startDate,
			endDate,
		).Return(test.actualSpend, nil)

		// Mock Usage service
		inputUsage := usage.Usage{
			PrincipalID:  "test-user",
			AccountID:    "",
			StartDate:    startDate.Unix(),
			EndDate:      usageEndDate.Unix(),
			CostAmount:   test.actualSpend,
			CostCurrency: "USD",
			TimeToLive:   startDate.AddDate(0, 1, 0).Unix(),
		}

		budgetStartTime := time.Unix(input.lease.LeaseStatusModifiedOn, 0)
		usageSvc.On("PutUsage", inputUsage).Return(nil)
		usageSvc.On("GetUsageByDateRange", budgetStartTime, usageEndDate.AddDate(0, 0, -1)).Return(nil, nil)

		// Should transition from "Active" --> "FinanceLock"
		if test.shouldTransitionLeaseStatus {
			dbSvc.On("TransitionLeaseStatus",
				"1234567890", "test-user",
				db.Active, test.expectedLeaseStatusTransition,
			).Return(func(acctID string, pID string, from db.LeaseStatus, to db.LeaseStatus) *db.DceLease {
				// Return the lease object, with it's updated status
				input.lease.LeaseStatus = test.expectedLeaseStatusTransition
				return input.lease
			}, test.transitionLeaseError)
		}

		// Should publish to `lease-locked` topic
		if test.shouldSNS {
			lockedLease := *input.lease
			lockedLease.LeaseStatus = db.FinanceLock
			lockedLeaseMsg, err := common.PrepareSNSMessageJSON(lockedLease)
			require.Nil(t, err)
			snsSvc.On("PublishMessage",
				aws.String("lease-locked"),
				&lockedLeaseMsg,
				true,
			).Return(&lockedLeaseMsg, test.snsError)
		}

		// Should add the account to the reset queue
		if test.shouldSQSReset {
			sqsSvc.On("SendMessage", &sqs.SendMessageInput{
				QueueUrl:    aws.String("reset-queue-url"),
				MessageBody: aws.String("1234567890"),
			}).Return(&sqs.SendMessageOutput{}, test.sqsError)
		}

		// Should send a notification email
		if test.shouldSendEmail {
			emailSvc.On("SendEmail", &email.SendEmailInput{
				FromAddress:  "from@example.com",
				ToAddresses:  []string{"recipA@example.com", "recipB@example.com"},
				BCCAddresses: []string{"bcc@example.com"},
				Subject:      test.expectedEmailSubject,
				BodyHTML:     test.expectedEmailBodyHTML,
				BodyText:     test.expectedEmailBodyText,
			}).Return(nil)
		}

		// Call Lambda handler
		err := lambdaHandler(input)
		if test.expectedError == "" {
			require.Nil(t, err)
		} else {
			require.Regexp(t, test.expectedError, err)
		}

		// Check we called our services
		dbSvc.AssertExpectations(t)
		tokenSvc.AssertExpectations(t)
		budgetSvc.AssertExpectations(t)
		snsSvc.AssertExpectations(t)
		sqsSvc.AssertExpectations(t)
		emailSvc.AssertExpectations(t)
	}

	t.Run("Scenario: Over Budget Lease", func(t *testing.T) {
		checkBudgetTest(&checkBudgetTestInput{
			// Over budget
			budgetAmount: 100,
			actualSpend:  150,
			// Should transition from Active --> FinanceLock
			leaseStatus:                   db.Active,
			expectedLeaseStatusTransition: db.FinanceLock,
			// Should do all the finance locking things
			shouldTransitionLeaseStatus: true,
			shouldSNS:                   true,
			shouldSQSReset:              true,
			// Should send notification email
			shouldSendEmail:       true,
			expectedEmailSubject:  expectedOverBudgetText,
			expectedEmailBodyHTML: expectedOverBudgetEmailHTML,
			expectedEmailBodyText: expectedOverBudgetEmailText,
		})
	})

	t.Run("Scenario: Over Threshold Lease", func(t *testing.T) {
		checkBudgetTest(&checkBudgetTestInput{
			// >75% of budget
			budgetAmount: 100,
			actualSpend:  76,
			leaseStatus:  db.Active,
			// Should not finance lock or reset
			shouldTransitionLeaseStatus: false,
			shouldSNS:                   false,
			shouldSQSReset:              false,
			// Should send notification email
			shouldSendEmail:      true,
			expectedEmailSubject: "AWS Dce Lease at 75% of budget [1234567890]",
			expectedEmailBodyHTML: strings.TrimSpace(`
<p>

AWS Dce Lease for principal test-user in AWS Account 1234567890
has exceeded the 75% threshold limit for its budget of $100.
Actual spend is $76

</p>
`),
			expectedEmailBodyText: strings.TrimSpace(`
AWS Dce Lease for principal test-user in AWS Account 1234567890
has exceeded the 75% threshold limit for its budget of $100.
Actual spend is $76
`),
		})
	})

	t.Run("Scenario: Under Budget Lease", func(t *testing.T) {
		checkBudgetTest(&checkBudgetTestInput{
			// <75% of budget
			budgetAmount: 100,
			actualSpend:  50,
			leaseStatus:  db.Active,
			// Should not finance lock or reset
			shouldTransitionLeaseStatus: false,
			shouldSNS:                   false,
			shouldSQSReset:              false,
			// Should not send notification email
			shouldSendEmail: false,
		})
	})

	t.Run("should handle errors and continue", func(t *testing.T) {
		// Continue if DB fails
		checkBudgetTest(&checkBudgetTestInput{
			// Over budget
			budgetAmount: 100,
			actualSpend:  150,

			// DB Transition fails
			leaseStatus:                   db.Active,
			expectedLeaseStatusTransition: db.FinanceLock,
			transitionLeaseError:          errors.New("DB transition failed"),

			// Should continue on error
			shouldTransitionLeaseStatus: true,
			shouldSNS:                   true,
			shouldSQSReset:              true,
			shouldSendEmail:             true,
			expectedEmailSubject:        expectedOverBudgetText,
			expectedEmailBodyHTML:       expectedOverBudgetEmailHTML,
			expectedEmailBodyText:       expectedOverBudgetEmailText,

			// Should return an error
			expectedError: "DB transition failed",
		})

		// Continue if SNS fails
		checkBudgetTest(&checkBudgetTestInput{
			// Over budget
			budgetAmount: 100,
			actualSpend:  150,

			leaseStatus:                   db.Active,
			expectedLeaseStatusTransition: db.FinanceLock,
			shouldTransitionLeaseStatus:   true,

			// SNS Fails
			shouldSNS: true,
			snsError:  errors.New("SNS failed"),

			// Should continue on error
			shouldSQSReset:        true,
			shouldSendEmail:       true,
			expectedEmailSubject:  expectedOverBudgetText,
			expectedEmailBodyHTML: expectedOverBudgetEmailHTML,
			expectedEmailBodyText: expectedOverBudgetEmailText,

			// Should return an error
			expectedError: "SNS failed",
		})

		// Continue if SQS fails
		checkBudgetTest(&checkBudgetTestInput{
			// Over budget
			budgetAmount: 100,
			actualSpend:  150,

			leaseStatus:                   db.Active,
			expectedLeaseStatusTransition: db.FinanceLock,
			shouldTransitionLeaseStatus:   true,
			shouldSNS:                     true,

			// SQS fails
			shouldSQSReset: true,
			sqsError:       errors.New("sqs error"),

			// Should continue on error
			shouldSendEmail:       true,
			expectedEmailSubject:  expectedOverBudgetText,
			expectedEmailBodyHTML: expectedOverBudgetEmailHTML,
			expectedEmailBodyText: expectedOverBudgetEmailText,

			// Should return an error
			expectedError: "sqs error",
		})

		// Multiple errors
		checkBudgetTest(&checkBudgetTestInput{
			// Over budget
			budgetAmount: 100,
			actualSpend:  150,

			leaseStatus:                   db.Active,
			expectedLeaseStatusTransition: db.FinanceLock,
			shouldTransitionLeaseStatus:   true,

			// SNS Fails
			shouldSNS: true,
			snsError:  errors.New("sns error"),

			// SQS fails
			shouldSQSReset: true,
			sqsError:       errors.New("sqs error"),

			// Should continue on error
			shouldSendEmail:       true,
			expectedEmailSubject:  expectedOverBudgetText,
			expectedEmailBodyHTML: expectedOverBudgetEmailHTML,
			expectedEmailBodyText: expectedOverBudgetEmailText,

			// Should return a combined error
			expectedError: "sns error.*sqs error",
		})
	})

}
