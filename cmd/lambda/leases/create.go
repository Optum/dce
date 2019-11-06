package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/provision"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-lambda-go/events"
	"log"
	"net/http"
)

// CreateController is responsible for handling API events for creating leases.
type CreateController struct {
	Dao                   db.DBer
	Provisioner           provision.Provisioner
	SNS                   common.Notificationer
	LeaseTopicARN         *string
	UsageSvc              usage.Service
	PrincipalBudgetAmount *float64
	PrincipalBudgetPeriod *string
	MaxLeaseBudgetAmount  *float64
	MaxLeasePeriod        *int
}

type createLeaseRequest struct {
	PrincipalID              string   `json:"principalId"`
	AccountID                string   `json:"accountId"`
	BudgetAmount             float64  `json:"budgetAmount"`
	BudgetCurrency           string   `json:"budgetCurrency"`
	BudgetNotificationEmails []string `json:"budgetNotificationEmails"`
	ExpiresOn                int64    `json:"expiresOn"`
}

// Call - Function to validate the account request to add into the pool and
// publish the account creation to its respective client
// This function returns both a proxy response and an error. In this case,
// if we know how to handle the error (such as a bad request), then the err
// returned is nil. It's only not nil if we get an error that we don't know
// what to do with, in which case the calling router will handle it.
func (c CreateController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Extract the Body from the Request
	requestBody, err := validateLeaseRequest(c, req)
	if err != nil {
		return response.BadRequestError(err.Error()), nil
	}

	principalID := requestBody.PrincipalID
	log.Printf("Provisioning Account for Principal %s", principalID)

	// Check if the principal has any existing Active/FinanceLock/ResetLock
	// Leases
	checkLease, err := c.Provisioner.FindActiveLeaseForPrincipal(principalID)
	if err != nil {
		log.Printf("Failed to Check Principal Active Leases: %s", err)
		return response.ServerErrorWithResponse(fmt.Sprintf("Failed to verify if Principal has an existing lease: %s",
			err)), nil
	} else if checkLease.PrincipalID == principalID {
		errStr := fmt.Sprintf("Principal already has an active lease: %s",
			checkLease.AccountID)
		log.Printf(errStr)
		return response.ConflictError(errStr), nil
	}
	log.Printf("Principal %s has no Active Leases\n", principalID)

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

	// Check if the Principal and Account has been leased before
	lease, err := c.Provisioner.FindLeaseWithAccount(principalID,
		account.ID)
	if err != nil {
		log.Printf("Failed to Check Leases with Account: %s", err)
		return response.ServerErrorWithResponse(fmt.Sprintf("Failed to lookup leases: %s", err)), nil
	}

	// Create/Update an Account Lease to Active
	create := lease.AccountID == ""
	lease, err = c.Provisioner.ActivateAccount(create, principalID,
		account.ID, requestBody.BudgetAmount, requestBody.BudgetCurrency, requestBody.BudgetNotificationEmails,
		requestBody.ExpiresOn)
	if err != nil {
		log.Printf("Failed to Activate Account Lease: %s", err)
		return response.ServerErrorWithResponse(fmt.Sprintf("Failed to Create Lease for Account : %s", account.ID)), nil
	}

	// Set the Account as leased
	log.Printf("Set Account %s Status to Leased for Principal %s\n", principalID,
		account.ID)
	_, err = c.Dao.TransitionAccountStatus(account.ID, db.Ready, db.Leased)
	if err != nil {
		// Rollback
		log.Printf("Error to Transition Account Status: %s", err)
		return rollbackProvision(c.Provisioner, err, false, principalID, account.ID), nil
	}

	// Publish Lease to the topic
	message, err := publishLease(c.SNS, lease, c.LeaseTopicARN)
	if err != nil {
		log.Printf("Error Publish Lease to Topic: %s", err)
		return rollbackProvision(c.Provisioner, err, true, principalID, account.ID), nil
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
	assgn *db.Lease, topic *string) (*string, error) {
	// Create a LeaseResponse based on the assgn
	assgnResp := response.CreateLeaseResponse(assgn)

	// Create the message to send to the topic from the Lease
	messageBytes, err := json.Marshal(assgnResp)
	if err != nil {
		// Rollback
		log.Printf("Error to Marshal Account Lease: %s", err)
		return nil, err
	}
	message := string(messageBytes)

	// Create the messageBody to make it compatible with SNS JSON
	provBody := messageBody{
		Default: message,
		Body:    message,
	}
	provMessageBytes, err := json.Marshal(provBody)
	if err != nil {
		// Rollback
		log.Printf("Error to Marshal Message Body: %s", err)
		return nil, err
	}
	provMessage := string(provMessageBytes)

	// Publish message to the Provision Topic on the success of the Account
	// Lease
	log.Printf("Sending Lease Message to SNS Topic %s\n", *topic)
	messageID, err := snsSvc.PublishMessage(topic, &provMessage, true)
	if err != nil {
		// Rollback
		log.Printf("Error to Send Message to SNS Topic %s: %s", *topic, err)
		return nil, err
	}
	log.Printf("Success Message Sent to SNS Topic %s: %s\n", *topic, *messageID)
	return &message, nil
}

// rollbackProvision is a helper function to execute rollback for account
// provisioning
func rollbackProvision(prov provision.Provisioner, err error,
	transitionAccountStatus bool, principalID string,
	accountID string) events.APIGatewayProxyResponse {
	// Attempt Rollback
	var message string
	errRollBack := prov.RollbackProvisionAccount(transitionAccountStatus,
		principalID, accountID)
	if errRollBack != nil {
		log.Printf("Error to Rollback: %s", errRollBack)
		message = fmt.Sprintf("Failed to Rollback "+
			"Account Lease for %s - %s", accountID, principalID)
	} else {
		message = fmt.Sprintf("Failed to Create "+
			"Lease for %s - %s", accountID, principalID)
	}

	// Return an error
	return response.ServerErrorWithResponse(string(message))
}
