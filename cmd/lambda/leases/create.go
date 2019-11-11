package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"time"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-lambda-go/events"
)

// CreateController is responsible for handling API events for creating leases.
type CreateController struct {
	Dao                      db.DBer
	SNS                      common.Notificationer
	LeaseAddedTopicARN       *string
	UsageSvc                 usage.Service
	PrincipalBudgetAmount    *float64
	PrincipalBudgetPeriod    *string
	MaxLeaseBudgetAmount     *float64
	MaxLeasePeriod           *int
	DefaultLeaseLengthInDays int
}

type createLeaseRequest struct {
	PrincipalID              string                 `json:"principalId"`
	BudgetAmount             float64                `json:"budgetAmount"`
	BudgetCurrency           string                 `json:"budgetCurrency"`
	BudgetNotificationEmails []string               `json:"budgetNotificationEmails"`
	ExpiresOn                int64                  `json:"expiresOn"`
	Metadata                 map[string]interface{} `json:"metadata"`
}

// Call - Function to validate the account request to add into the pool and
// publish the account creation to its respective client
// This function returns both a proxy response and an error. In this case,
// if we know how to handle the error (such as a bad request), then the err
// returned is nil. It's only not nil if we get an error that we don't know
// what to do with, in which case the calling router will handle it.
func (c CreateController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Extract the Body from the Request
	requestBody, isValid, validationErrorMessage, err := validateLeaseRequest(c, req)

	if err != nil {
		return response.ServerErrorWithResponse(err.Error()), nil
	}

	if !isValid {
		return response.RequestValidationError(validationErrorMessage), nil
	}

	principalID := requestBody.PrincipalID
	log.Printf("Creating lease for Principal %s", principalID)

	// Fail if the Principal already has an active lease
	principalLeases, err := c.Dao.FindLeasesByPrincipal(requestBody.PrincipalID)
	if err != nil {
		log.Printf("Failed to list leases for principal %s: %s", requestBody.PrincipalID, err)
		return response.ServerError(), nil
	}
	if principalLeases != nil {
		for _, lease := range principalLeases {
			if lease.LeaseStatus == db.Active {
				msg := fmt.Sprintf("Principal already has an active lease for account %s", lease.AccountID)
				return response.ConflictError(msg), nil
			}
		}
	}

	// Get the First Ready Account
	// Exit if there's an error or no ready accounts
	account, err := c.Dao.GetReadyAccount()
	if err != nil {
		log.Printf("Failed to Check Ready Accounts: %s", err)
		return response.ServerErrorWithResponse(
			fmt.Sprintf("Failed to find a Ready Account: %s", err)), nil
	} else if account == nil {
		errStr := "No Available accounts at this moment"
		log.Printf(errStr)
		return response.ServiceUnavailableError(errStr), nil
	}
	log.Printf("Principal %s will be Leased to Account: %s\n", principalID,
		account.ID)

	// Create/Update lease record
	now := time.Now()
	lease, err := c.Dao.UpsertLease(db.Lease{
		AccountID:                account.ID,
		PrincipalID:              requestBody.PrincipalID,
		ID:                       uuid.New().String(),
		LeaseStatus:              db.Active,
		LeaseStatusReason:        db.LeaseActive,
		BudgetAmount:             requestBody.BudgetAmount,
		BudgetCurrency:           requestBody.BudgetCurrency,
		BudgetNotificationEmails: requestBody.BudgetNotificationEmails,
		CreatedOn:                now.Unix(),
		LastModifiedOn:           now.Unix(),
		LeaseStatusModifiedOn:    now.Unix(),
		ExpiresOn:                requestBody.ExpiresOn,
		Metadata:                 requestBody.Metadata,
	})
	if err != nil {
		log.Printf("Failed to create lease DB record for %s @ %s: %s",
			requestBody.PrincipalID, account.ID, err)
		return response.ServerError(), nil
	}

	// Mark the account as Status=Leased
	account, err = c.Dao.TransitionAccountStatus(account.ID, db.Ready, db.Leased)
	if err != nil {
		log.Printf("ERROR Failed to transition account %s to Leased for lease for %s",
			lease.AccountID, lease.PrincipalID)
		return response.ServerError(), nil
	}

	// Publish Lease to the topic
	message, err := publishLease(c.SNS, lease, c.LeaseAddedTopicARN)
	if err != nil {
		log.Print(err.Error())
		return response.ServerError(), nil
	}

	// Return the response back to API
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Body:       *message,
	}, nil
}

// publishLease is a helper function to create and publish an lease
// structured message to an SNS Topic
func publishLease(snsSvc common.Notificationer,
	lease *db.Lease, topic *string) (*string, error) {
	// Create a LeaseResponse based on the lease
	leaseResp := response.CreateLeaseResponse(lease)

	// Create the message to send to the topic from the Lease
	messageBytes, err := json.Marshal(leaseResp)
	if err != nil {
		// Rollback
		log.Printf("Error to Marshal Account Lease: %s", err)
		return nil, err
	}
	message := string(messageBytes)

	// Create the messageBody to make it compatible with SNS JSON
	leaseMsgBody := messageBody{
		Default: message,
		Body:    message,
	}
	leaseMsgBytes, err := json.Marshal(leaseMsgBody)
	if err != nil {
		// Rollback
		log.Printf("Error to Marshal Message Body: %s", err)
		return nil, err
	}
	leaseMsg := string(leaseMsgBytes)

	// Publish message to the lease topic on the success of the lease creation
	log.Printf("Sending Lease Message to SNS Topic %s\n", *topic)
	messageID, err := snsSvc.PublishMessage(topic, &leaseMsg, true)
	if err != nil {
		return nil, errors.Wrapf(err, "Error to Send Message to SNS Topic %s", *topic)
	}
	log.Printf("Success Message Sent to SNS Topic %s: %s\n", *topic, *messageID)
	return &message, nil
}
