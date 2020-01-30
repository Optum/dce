package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	awsMocks "github.com/Optum/dce/pkg/awsiface/mocks"
	budgetMocks "github.com/Optum/dce/pkg/budget/mocks"
	commonMocks "github.com/Optum/dce/pkg/common/mocks"
	"github.com/Optum/dce/pkg/db"
	dbMocks "github.com/Optum/dce/pkg/db/mocks"
	"github.com/Optum/dce/pkg/email"
	emailMocks "github.com/Optum/dce/pkg/email/mocks"
	"github.com/Optum/dce/pkg/usage"
	usageMocks "github.com/Optum/dce/pkg/usage/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCheckBudget(t *testing.T) {

	// Configure email templates
	emailTemplateHTML := `
<p>
{{if .IsOverBudget}}
Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded its budget of ${{.Lease.BudgetAmount}}. Actual spend is ${{.ActualSpend}}
{{else}}
Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded the {{.ThresholdPercentile}}% threshold limit for its budget of ${{.Lease.BudgetAmount}}.
Actual spend is ${{.ActualSpend}}
{{end}}
</p>
`
	emailTemplateText := `
{{if .IsOverBudget}}
Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded its budget of ${{.Lease.BudgetAmount}}. Actual spend is ${{.ActualSpend}}
{{else}}
Lease for principal {{.Lease.PrincipalID}} in AWS Account {{.Lease.AccountID}}
has exceeded the {{.ThresholdPercentile}}% threshold limit for its budget of ${{.Lease.BudgetAmount}}.
Actual spend is ${{.ActualSpend}}
{{end}}
`
	emailTemplateSubject := `
Lease {{if .IsOverBudget}}over budget{{else}}at {{.ThresholdPercentile}}% of budget{{end}} [{{.Lease.AccountID}}]
`

	expectedOverBudgetEmailHTML := strings.TrimSpace(`
<p>

Lease for principal test-user in AWS Account 1234567890
has exceeded its budget of $100. Actual spend is $150

</p>
`)
	expectedOverBudgetEmailText := strings.TrimSpace(`
Lease for principal test-user in AWS Account 1234567890
has exceeded its budget of $100. Actual spend is $150
`)
	expectedOverBudgetText := "Lease over budget [1234567890]"

	type checkBudgetTestInput struct {
		budgetAmount                  float64
		actualSpend                   float64
		leaseStatus                   db.LeaseStatus
		expectedLeaseStatusTransition db.LeaseStatus
		shouldTransitionLeaseStatus   bool
		transitionLeaseError          error
		shouldSNS                     bool
		shouldSQSReset                bool
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
			lease: &db.Lease{
				AccountID:                "1234567890",
				PrincipalID:              "test-user",
				LeaseStatus:              test.leaseStatus,
				BudgetAmount:             test.budgetAmount,
				BudgetCurrency:           "USD",
				BudgetNotificationEmails: []string{"recipA@example.com", "recipB@example.com"},
				LeaseStatusModifiedOn:    time.Unix(100, 0).Unix(),
				ExpiresOn:                time.Now().AddDate(0, 0, +1000).Unix(), //Make sure it expires in the distant future as we aren't testing that
			},
			awsSession:                             &awsMocks.AwsSession{},
			tokenSvc:                               tokenSvc,
			budgetSvc:                              budgetSvc,
			usageSvc:                               usageSvc,
			snsSvc:                                 snsSvc,
			leaseLockedTopicArn:                    "lease-locked",
			sqsSvc:                                 sqsSvc,
			emailSvc:                               emailSvc,
			budgetNotificationFromEmail:            "from@example.com",
			budgetNotificationBCCEmails:            []string{"bcc@example.com"},
			budgetNotificationTemplateHTML:         emailTemplateHTML,
			budgetNotificationTemplateText:         emailTemplateText,
			budgetNotificationTemplateSubject:      emailTemplateSubject,
			budgetNotificationThresholdPercentiles: []float64{75, 100},
			principalBudgetAmount:                  1000,
		}

		// Should grab the account from the DB, to get it's adminRoleArn
		dbSvc.On("GetAccount", "1234567890").
			Return(&db.Account{
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
		usageSvc.On("GetUsageByDateRange", mock.Anything, mock.Anything).Return(nil, nil)

		// Should transition from "Active" --> "FinanceLock"
		if test.shouldTransitionLeaseStatus {
			dbSvc.On("TransitionLeaseStatus",
				"1234567890", "test-user",
				db.Active, test.expectedLeaseStatusTransition,
				mock.Anything,
			).Return(func(acctID string, pID string, from db.LeaseStatus, to db.LeaseStatus, reason db.LeaseStatusReason) *db.Lease {
				// Return the lease object, with it's updated status
				input.lease.LeaseStatus = test.expectedLeaseStatusTransition
				return input.lease
			}, test.transitionLeaseError)
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
			expectedLeaseStatusTransition: db.Inactive,
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
			expectedEmailSubject: "Lease at 75% of budget [1234567890]",
			expectedEmailBodyHTML: strings.TrimSpace(`
<p>

Lease for principal test-user in AWS Account 1234567890
has exceeded the 75% threshold limit for its budget of $100.
Actual spend is $76

</p>
`),
			expectedEmailBodyText: strings.TrimSpace(`
Lease for principal test-user in AWS Account 1234567890
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
			expectedLeaseStatusTransition: db.Inactive,
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

	})
}
func Test_isLeaseExpired(t *testing.T) {
	type args struct {
		lease                *db.Lease
		context              *leaseContext
		actualPrincipalSpend float64
	}
	emails := []string{"joe@example.com"}
	principalBudgetAmount := 7000.00
	lease := &db.Lease{
		AccountID:                "12345",
		PrincipalID:              "98765",
		LeaseStatus:              db.Inactive,
		LeaseStatusReason:        db.LeaseExpired,
		CreatedOn:                time.Now().Unix(),
		LastModifiedOn:           time.Now().Unix(),
		BudgetAmount:             3000,
		BudgetCurrency:           "USD",
		BudgetNotificationEmails: emails,
		LeaseStatusModifiedOn:    time.Now().Unix(),
		ExpiresOn:                time.Now().Unix()}

	nonExpiredLeaseTestArgs := &args{
		lease,
		&leaseContext{
			time.Now().AddDate(0, 0, -1).Unix(),
			10},
		10}

	expiredLeaseTestArgs := &args{
		lease,
		&leaseContext{
			time.Now().AddDate(0, 0, +1).Unix(),
			10},
		10}

	overBudgetTest := &args{
		lease,
		&leaseContext{
			time.Now().AddDate(0, 0, -1).Unix(),
			5000},
		5000}

	overPrincipalBudgetAmountTest := &args{
		lease,
		&leaseContext{
			time.Now().AddDate(0, 0, -1).Unix(),
			2500},
		9000}

	tests := []struct {
		name  string
		args  args
		want  bool
		want1 db.LeaseStatusReason
	}{
		{"Non-expired lease test", *nonExpiredLeaseTestArgs, false, db.LeaseActive},
		{"Expired lease test", *expiredLeaseTestArgs, true, db.LeaseExpired},
		{"Over budget lease test", *overBudgetTest, true, db.LeaseOverBudget},
		{"Over principal budget amount test", *overPrincipalBudgetAmountTest, true, db.LeaseOverPrincipalBudget},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := isLeaseExpired(tt.args.lease, tt.args.context, tt.args.actualPrincipalSpend, principalBudgetAmount)
			if got != tt.want {
				t.Errorf("isLeaseExpired() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("isLeaseExpired() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestGetBeginningOfCurrentBillingPeriod(t *testing.T) {

	actualOutput := getBeginningOfCurrentBillingPeriod("WEEKLY")

	currentTime := time.Now()
	for currentTime.Weekday() != time.Sunday { // iterate back to Sunday
		currentTime = currentTime.AddDate(0, 0, -1)
	}

	expectedOutput := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC)

	require.NotNil(t, actualOutput)
	require.Equal(t, expectedOutput, actualOutput)
}
