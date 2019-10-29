package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/provision"
	"github.com/aws/aws-lambda-go/events"
)

type CreateLeaseRequest struct {
	PrincipalID              string   `json:"principalId"`
	AccountID                string   `json:"accountId"`
	BudgetAmount             float64  `json:"budgetAmount"`
	BudgetCurrency           string   `json:"budgetCurrency"`
	BudgetNotificationEmails []string `json:"budgetNotificationEmails"`
	ExpiresOn                int64    `json:"expiresOn"`
}

// CreateLease - Function to validate the account request to add into the pool and
// publish the account creation to its respective client
// This function returns both a proxy response and an error. In this case,
// if we know how to handle the error (such as a bad request), then the err
// returned is nil. It's only not nil if we get an error that we don't know
// what to do with, in which case the calling router will handle it.
func CreateLease(w http.ResponseWriter, r *http.Request) {

	// Extract the Body from the Request
	requestBody := &CreateLeaseRequest{}
	var err error
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&requestBody)
	if err != nil || requestBody.PrincipalID == "" {
		log.Printf("Failed to Parse Request Body: %s", r.Body)
		ServerErrorWithResponse(w, fmt.Sprintf("Failed to Parse Request Body: %s", r.Body))
	}

	principalID := requestBody.PrincipalID
	log.Printf("Provisioning Account for Principal %s", principalID)

	// Just do a quick sanity check on the request and make sure that the
	// requested lease end date, if specified, is at least greater than
	// today and if it isn't then return an error response
	if requestBody.ExpiresOn != 0 && requestBody.ExpiresOn <= time.Now().Unix() {
		errStr := fmt.Sprintf("Requested lease has a desired expiry date less than today: %d", requestBody.ExpiresOn)
		log.Printf(errStr)
		ServerErrorWithResponse(w, fmt.Sprintf("Failed to Parse Request Body: %s", r.Body))
		// return response.BadRequestError(errStr), nil
	}

	// Check if the principal has any existing Active/FinanceLock/ResetLock
	// Leases
	checkLease, err := Provisioner.FindActiveLeaseForPrincipal(principalID)
	if err != nil {
		log.Printf("Failed to Check Principal Active Leases: %s", err)
		ServerErrorWithResponse(w, fmt.Sprintf("Cannot verify if Principal has existing Redbox Account : %s", err))
	} else if checkLease.PrincipalID == principalID {
		errStr := fmt.Sprintf("Principal already has an existing Redbox: %s",
			checkLease.AccountID)
		log.Printf(errStr)
		ServerErrorWithResponse(w, fmt.Sprintf("Failed to Parse Request Body: %s", r.Body))
		// return response.ConflictError(errStr), nil
	}
	log.Printf("Principal %s has no Active Leases\n", principalID)

	// Get the First Ready Account
	// Exit if there's an error or no ready accounts
	account, err := DbSvc.GetReadyAccount()
	if err != nil {
		log.Printf("Failed to Check Ready Accounts: %s", err)
		ServerErrorWithResponse(w, fmt.Sprintf("Failed to Parse Request Body: %s", r.Body))
		// return response.ServerErrorWithResponse(
		//	fmt.Sprintf("Cannot get Available Redbox Accounts : %s", err)), nil
	} else if account == nil {
		errStr := "No Available Redbox Accounts at this moment"
		log.Printf(errStr)
		ServerErrorWithResponse(w, fmt.Sprintf("Failed to Parse Request Body: %s", r.Body))
		// return response.ServiceUnavailableError(errStr), nil
	}
	log.Printf("Principal %s will be Leased to Account: %s\n", principalID,
		account.ID)

	// Check if the Principal and Account has been leased before
	lease, err := Provisioner.FindLeaseWithAccount(principalID,
		account.ID)
	if err != nil {
		log.Printf("Failed to Check Leases with Account: %s", err)
		ServerErrorWithResponse(w, fmt.Sprintf("Failed to Parse Request Body: %s", r.Body))
		// return response.ServerErrorWithResponse(fmt.Sprintf("Cannot get Available Redbox Accounts : %s", err)), nil
	}

	// Create/Update a Redbox Account Lease to Active
	create := lease.AccountID == ""
	lease, err = Provisioner.ActivateAccount(create, principalID,
		account.ID, requestBody.BudgetAmount, requestBody.BudgetCurrency, requestBody.BudgetNotificationEmails,
		requestBody.ExpiresOn)
	if err != nil {
		log.Printf("Failed to Activate Account Lease: %s", err)
		ServerErrorWithResponse(w, fmt.Sprintf("Failed to Parse Request Body: %s", r.Body))
		// return response.ServerErrorWithResponse(fmt.Sprintf("Failed to Create Lease for Account : %s", account.ID)), nil
	}

	// Set the Account as leased
	log.Printf("Set Account %s Status to Leased for Principal %s\n", principalID,
		account.ID)
	_, err = DbSvc.TransitionAccountStatus(account.ID, db.Ready, db.Leased)
	if err != nil {
		// Rollback
		log.Printf("Error to Transition Account Status: %s", err)
		ServerErrorWithResponse(w, fmt.Sprintf("Failed to Parse Request Body: %s", r.Body))
		// return rollbackProvision(c.Provisioner, err, false, principalID, account.ID), nil
	}

	// Publish Lease to the topic
	message, err := publishLease(SnsSvc, lease, ProvisionTopicArn)
	if err != nil {
		log.Printf("Error Publish Lease to Topic: %s", err)
		ServerErrorWithResponse(w, fmt.Sprintf("Failed to Parse Request Body: %s", r.Body))
		// return rollbackProvision(c.Provisioner, err, true, principalID, account.ID), nil
	}

	json.NewEncoder(w).Encode(*message)
}

// publishLease is a helper function to create and publish an lease
// structured message to an SNS Topic
func publishLease(snsSvc common.Notificationer,
	assgn *db.RedboxLease, topic *string) (*string, error) {
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
