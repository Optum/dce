package main

import (
	"encoding/json"
	"fmt"
	"github.com/Optum/Redbox/pkg/awsiface"
	"github.com/Optum/Redbox/pkg/budget"
	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/email"
	multierrors "github.com/Optum/Redbox/pkg/errors"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/pkg/errors"
	"log"
)

func main() {
	lambda.Start(func(event interface{}) {
		log.Printf("Initializing budget check")

		// Cast the event as a Lease object
		lease, err := eventToLease(event)
		if err != nil {
			log.Fatalf("Invalid lambda event: %s. Expected a Lease object, received: %v", err, event)
		}
		log.Printf("Checking budget for lease %s @ %s", lease.PrincipalID, lease.AccountID)

		// Configure the DB service
		dbSvc, err := db.NewFromEnv()
		if err != nil {
			log.Fatalf("Failed to configure DB service %s", err)
		}

		// Configure the STS Token service
		awsSession := session.Must(session.NewSession())
		stsSvc := sts.New(awsSession)
		tokenSvc := &common.STS{Client: stsSvc}

		err = lambdaHandler(&lambdaHandlerInput{
			dbSvc:                                  dbSvc,
			lease:                                  lease,
			awsSession:                             awsSession,
			tokenSvc:                               tokenSvc,
			budgetSvc:                              &budget.AWSBudgetService{},
			sqsSvc:                                 sqs.New(awsSession),
			resetQueueURL:                          common.RequireEnv("RESET_QUEUE_URL"),
			snsSvc:                                 &common.SNS{Client: sns.New(awsSession)},
			leaseLockedTopicArn:                    common.RequireEnv("LEASE_LOCKED_TOPIC_ARN"),
			emailSvc:                               &email.SESEmailService{SES: ses.New(awsSession)},
			budgetNotificationFromEmail:            common.RequireEnv("BUDGET_NOTIFICATION_FROM_EMAIL"),
			budgetNotificationBCCEmails:            common.RequireEnvStringSlice("BUDGET_NOTIFICATION_BCC_EMAILS", ","),
			budgetNotificationTemplateHTML:         common.RequireEnv("BUDGET_NOTIFICATION_TEMPLATE_HTML"),
			budgetNotificationTemplateText:         common.RequireEnv("BUDGET_NOTIFICATION_TEMPLATE_TEXT"),
			budgetNotificationTemplateSubject:      common.RequireEnv("BUDGET_NOTIFICATION_TEMPLATE_SUBJECT"),
			budgetNotificationThresholdPercentiles: common.RequireEnvFloatSlice("BUDGET_NOTIFICATION_THRESHOLD_PERCENTILES", ","),
		})
		if err != nil {
			log.Fatalf("Failed check budget: %s", err)
		}

		log.Printf("Budget check for lease %s @ %s complete.", lease.PrincipalID, lease.AccountID)
	})
}

func eventToLease(leaseEvent interface{}) (*db.RedboxLease, error) {
	// Convert the interface to JSON
	mapJSON, err := json.Marshal(leaseEvent)
	if err != nil {
		return nil, err
	}

	// Convert the JSON back into a lease
	var lease db.RedboxLease
	err = json.Unmarshal(mapJSON, &lease)
	if err != nil {
		return nil, err
	}
	return &lease, nil
}

type lambdaHandlerInput struct {
	dbSvc                                  db.DBer
	lease                                  *db.RedboxLease
	awsSession                             awsiface.AwsSession
	tokenSvc                               common.TokenService
	budgetSvc                              budget.Service
	snsSvc                                 common.Notificationer
	leaseLockedTopicArn                    string
	sqsSvc                                 awsiface.SQSAPI
	resetQueueURL                          string
	emailSvc                               email.Service
	budgetNotificationFromEmail            string
	budgetNotificationBCCEmails            []string
	budgetNotificationTemplateHTML         string
	budgetNotificationTemplateText         string
	budgetNotificationTemplateSubject      string
	budgetNotificationThresholdPercentiles []float64
}

func lambdaHandler(input *lambdaHandlerInput) error {
	leaseLogID := fmt.Sprintf("%s @ %s", input.lease.PrincipalID, input.lease.PrincipalID)

	// Lookup the account for this lease,
	// so we can get the adminRoleArn
	account, err := input.dbSvc.GetAccount(input.lease.AccountID)
	if err != nil {
		return errors.Wrapf(err, "Failed to lookup account for lease %s", leaseLogID)
	}
	if account == nil {
		return fmt.Errorf("Account %s does not exist for principal %s",
			input.lease.AccountID, input.lease.PrincipalID)
	}

	// Calculate actual spend for the account
	actualSpend, err := calculateSpend(&calculateSpendInput{
		account:    account,
		lease:      input.lease,
		tokenSvc:   input.tokenSvc,
		budgetSvc:  input.budgetSvc,
		awsSession: input.awsSession,
	})
	if err != nil {
		return errors.Wrapf(err, "Failed to calculate spend for lease %s", leaseLogID)
	}
	// Defer errors until the end, so we
	// can continue on error
	deferredErrors := []error{}

	// Handle the over-budget case (finance lock, etc)
	if actualSpend >= input.lease.BudgetAmount {
		log.Printf("Lease %s is over budget ($%.2f / $%.2f). Locking account...", leaseLogID, actualSpend, input.lease.BudgetAmount)
		err := handleOverBudget(input)
		if err != nil {
			deferredErrors = append(deferredErrors, err)
		}
	}

	// Send notification emails, for budget thresholds
	err = sendBudgetNotificationEmail(&sendBudgetNotificationEmailInput{
		lease:                                  input.lease,
		emailSvc:                               input.emailSvc,
		budgetNotificationFromEmail:            input.budgetNotificationFromEmail,
		budgetNotificationBCCEmails:            input.budgetNotificationBCCEmails,
		budgetNotificationTemplateHTML:         input.budgetNotificationTemplateHTML,
		budgetNotificationTemplateText:         input.budgetNotificationTemplateText,
		budgetNotificationTemplateSubject:      input.budgetNotificationTemplateSubject,
		budgetNotificationThresholdPercentiles: input.budgetNotificationThresholdPercentiles,
		actualSpend:                            actualSpend,
	})
	if err != nil {
		log.Printf("Failed to send budget notification emails for lease %s @ %s: %s",
			input.lease.PrincipalID, input.lease.AccountID, err)
		deferredErrors = append(deferredErrors, err)
	}

	// Return deferred errors
	if len(deferredErrors) > 0 {
		return multierrors.NewMultiError("Budget check failed: ", deferredErrors)
	}

	return nil
}

// handleOverBudget handles the case where a lease is over budget:
// - Sets Lease DB status to FinanceLocked
// - Publish Lease to "lease-locked" SNS topic
// - Pushes account to reset queue (to stop the bleeding)
func handleOverBudget(input *lambdaHandlerInput) error {
	// Defer errors until the end, so we
	// can continue on error
	deferredErrors := []error{}

	// Set Lease.LeaseStatus = FinanceLock
	prevStatus := input.lease.LeaseStatus
	err := financeLock(&financeLockInput{
		lease: input.lease,
		dbSvc: input.dbSvc,
	})
	if err != nil {
		deferredErrors = append(deferredErrors, err)
	}

	// Publish a message to the "lease-locked" SNS topic
	// (only if the account went from Active --> Locked),
	if prevStatus != db.Active {
		log.Printf("Skipping publish to lease-locked SNS topic (lease was not previously active)")

	} else {
		log.Printf("Publishing to the lease-locked SNS topic")
		err := publishLeaseLocked(&publishLeaseLockedInput{
			snsSvc:              input.snsSvc,
			leaseLockedTopicArn: input.leaseLockedTopicArn,
			lease:               input.lease,
		})
		if err != nil {
			deferredErrors = append(deferredErrors, err)
		}
	}

	// Add the account to the SQS reset queue
	log.Printf("Adding account %s to the reset queue", input.lease.AccountID)
	_, err = input.sqsSvc.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    aws.String(input.resetQueueURL),
		MessageBody: aws.String(input.lease.AccountID),
	})
	if err != nil {
		log.Printf("Failed to add account to reset queue for lease %s @ %s: %s", input.lease.PrincipalID, input.lease.AccountID, err)
		deferredErrors = append(deferredErrors, err)
	}

	// Return errors
	if len(deferredErrors) > 0 {
		return multierrors.NewMultiError("Failed to lock over-budget account: ", deferredErrors)
	}

	return nil
}
