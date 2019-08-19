package main

import (
	"errors"
	awsMocks "github.com/Optum/Redbox/pkg/awsiface/mocks"
	budgetMocks "github.com/Optum/Redbox/pkg/budget/mocks"
	"github.com/Optum/Redbox/pkg/common"
	commonMocks "github.com/Optum/Redbox/pkg/common/mocks"
	"github.com/Optum/Redbox/pkg/db"
	dbMocks "github.com/Optum/Redbox/pkg/db/mocks"
	"github.com/Optum/Redbox/pkg/email"
	emailMocks "github.com/Optum/Redbox/pkg/email/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestCheckBudget(t *testing.T) {

	// Configure email templates
	emailTemplateHTML := `
<p>
{{if .IsOverBudget}}
AWS Redbox Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded its budget of ${{.Lease.BudgetAmount}}. Actual spend is ${{.ActualSpend}}
{{else}}
AWS Redbox Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded the {{.ThresholdPercentile}}% threshold limit for its budget of ${{.Lease.BudgetAmount}}.
Actual spend is ${{.ActualSpend}}
{{end}}
</p>
`
	emailTemplateText := `
{{if .IsOverBudget}}
AWS Redbox Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded its budget of ${{.Lease.BudgetAmount}}. Actual spend is ${{.ActualSpend}}
{{else}}
AWS Redbox Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded the {{.ThresholdPercentile}}% threshold limit for its budget of ${{.Lease.BudgetAmount}}.
Actual spend is ${{.ActualSpend}}
{{end}}
`
	emailTemplateSubject := `
AWS Redbox Lease {{if .IsOverBudget}}over budget{{else}}at {{.ThresholdPercentile}}% of budget{{end}} [{{.Lease.AccountID}}]
`

	expectedOverBudgetEmailHTML := strings.TrimSpace(`
<p>

AWS Redbox Lease for principal test-user in AWS Account 1234567890
has exceeded its budget of $100. Actual spend is $150

</p>
`)
	expectedOverBudgetEmailText := strings.TrimSpace(`
AWS Redbox Lease for principal test-user in AWS Account 1234567890
has exceeded its budget of $100. Actual spend is $150
`)
	expectedOverBudgetText := "AWS Redbox Lease over budget [1234567890]"

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
		snsSvc := &commonMocks.Notificationer{}
		sqsSvc := &awsMocks.SQSAPI{}
		emailSvc := &emailMocks.Service{}
		input := &lambdaHandlerInput{
			dbSvc: dbSvc,
			lease: &db.RedboxLease{
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
			Return(&db.RedboxAccount{
				AdminRoleArn: "mock:admin:role:arn",
			}, nil)

		// Mock the TokenService
		// Should assume Account.AdminRoleArn
		tokenSvc.MockNewSession("mock:admin:role:arn")

		// Mock the BudgetService, actualSpend=150 (over budget)
		// Should use assumed role
		budgetSvc.On("SetCostExplorer", mock.Anything)
		budgetSvc.On("CalculateTotalSpend",
			// Start time should be Lease.LeaseStatusModifiedOn
			// (which should match when it became active)
			time.Unix(100, 0),
			// End time should be ~tomorrow
			// (CostExplorer endTime is exclusive,
			//  so if we want costs for today, then endTime=tomorrow)
			mock.MatchedBy(func(endTime time.Time) bool {
				tomorrow := time.Now().Add(time.Hour * 24).Unix()
				assert.InDelta(t, tomorrow, endTime.Unix(), 2,
					"End time should be tomorrow")
				return true
			}),
		).Return(test.actualSpend, nil)

		// Should transition from "Active" --> "FinanceLock"
		if test.shouldTransitionLeaseStatus {
			dbSvc.On("TransitionLeaseStatus",
				"1234567890", "test-user",
				db.Active, test.expectedLeaseStatusTransition,
			).Return(func(acctID string, pID string, from db.LeaseStatus, to db.LeaseStatus) *db.RedboxLease {
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
			expectedEmailSubject: "AWS Redbox Lease at 75% of budget [1234567890]",
			expectedEmailBodyHTML: strings.TrimSpace(`
<p>

AWS Redbox Lease for principal test-user in AWS Account 1234567890
has exceeded the 75% threshold limit for its budget of $100.
Actual spend is $76

</p>
`),
			expectedEmailBodyText: strings.TrimSpace(`
AWS Redbox Lease for principal test-user in AWS Account 1234567890
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
