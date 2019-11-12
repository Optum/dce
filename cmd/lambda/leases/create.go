package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/provision"
)

var (
	provisioner           provision.AccountProvision
	leaseTopicARN         string
	principalBudgetAmount float64
	principalBudgetPeriod string
	maxLeaseBudgetAmount  float64
	maxLeasePeriod        int
)

func init() {
	provisioner = provision.AccountProvision{
		DBSvc: Dao,
	}

	leaseTopicARN = Config.GetEnvVar("PROVISION_TOPIC", "DCEDefaultProvisionTopic")
	principalBudgetAmount = Config.GetEnvFloatVar("PRINCIPAL_BUDGET_AMOUNT", 25.00)
	principalBudgetPeriod = Config.GetEnvVar("PRINCIPAL_BUDGET_PERIOD", Weekly)
	maxLeaseBudgetAmount = Config.GetEnvFloatVar("MAX_LEASE_BUDGET_AMOUNT", 25.00)
	maxLeasePeriod = Config.GetEnvIntVar("MAX_LEASE_PERIOD", 7)

}

type createLeaseRequest struct {
	PrincipalID              string   `json:"principalId"`
	AccountID                string   `json:"accountId"`
	BudgetAmount             float64  `json:"budgetAmount"`
	BudgetCurrency           string   `json:"budgetCurrency"`
	BudgetNotificationEmails []string `json:"budgetNotificationEmails"`
	ExpiresOn                int64    `json:"expiresOn"`
}

// CreateLease - Creates the lease
func CreateLease(w http.ResponseWriter, r *http.Request) {

	c := leaseValidationContext{}
	// Extract the Body from the Request
	requestBody, isValid, validationErrorMessage, err := validateLeaseFromRequest(&c, r)

	if err != nil {
		response.WriteServerErrorWithResponse(w, err.Error())
		return
	}

	if !isValid {
		response.WriteBadRequestError(w, validationErrorMessage)
		return
	}

	principalID := requestBody.PrincipalID
	log.Printf("Provisioning Account for Principal %s", principalID)

	// Check if the principal has any existing Active/FinanceLock/ResetLock
	// Leases
	checkLease, err := provisioner.FindActiveLeaseForPrincipal(principalID)
	if err != nil {
		log.Printf("Failed to Check Principal Active Leases: %s", err)
		response.WriteServerErrorWithResponse(w,
			fmt.Sprintf("Failed to verify if Principal has an existing lease: %s", err),
		)
		return
	} else if checkLease.PrincipalID == principalID {
		errStr := fmt.Sprintf("Principal already has an active lease: %s",
			checkLease.AccountID)
		log.Printf(errStr)
		response.WriteConflictError(w, errStr)
		return
	}
	log.Printf("Principal %s has no Active Leases\n", principalID)

	// Get the First Ready Account
	// Exit if there's an error or no ready accounts
	account, err := Dao.GetReadyAccount()
	if err != nil {
		log.Printf("Failed to Check Ready Accounts: %s", err)
		response.WriteServerErrorWithResponse(
			w,
			fmt.Sprintf("Failed to find a Ready Account: %s", err),
		)
		return
	} else if account == nil {
		errStr := "No Available accounts at this moment"
		log.Printf(errStr)
		response.WriteServiceUnavailableError(w, errStr)
		return
	}
	log.Printf("Principal %s will be Leased to Account: %s\n", principalID,
		account.ID)

	// Check if the Principal and Account has been leased before
	lease, err := provisioner.FindLeaseWithAccount(principalID,
		account.ID)
	if err != nil {
		log.Printf("Failed to Check Leases with Account: %s", err)
		response.WriteServerErrorWithResponse(w, fmt.Sprintf("Failed to lookup leases: %s", err))
		return
	}

	// Create/Update an Account Lease to Active
	create := lease.AccountID == ""
	lease, err = provisioner.ActivateAccount(create, principalID,
		account.ID, requestBody.BudgetAmount, requestBody.BudgetCurrency, requestBody.BudgetNotificationEmails,
		requestBody.ExpiresOn)
	if err != nil {
		log.Printf("Failed to Activate Account Lease: %s", err)
		response.WriteServerErrorWithResponse(w, fmt.Sprintf("Failed to Create Lease for Account : %s", account.ID))
		return
	}

	// Set the Account as leased
	log.Printf("Set Account %s Status to Leased for Principal %s\n", principalID,
		account.ID)
	_, err = Dao.TransitionAccountStatus(account.ID, db.Ready, db.Leased)
	if err != nil {
		// Rollback
		log.Printf("Error to Transition Account Status: %s", err)
		rollbackProvision(w, &provisioner, err, false, principalID, account.ID)
		return
	}

	// Publish Lease to the topic
	message, err := publishLease(SnsSvc, lease, &leaseTopicARN)
	if err != nil {
		log.Printf("Error Publish Lease to Topic: %s", err)
		rollbackProvision(w, &provisioner, err, true, principalID, account.ID)
		return
	}

	response.WriteAPIResponse(w, http.StatusCreated, *message)
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
func rollbackProvision(w http.ResponseWriter, prov provision.Provisioner, err error,
	transitionAccountStatus bool, principalID string,
	accountID string) {
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
	response.WriteServerErrorWithResponse(w, string(message))
}
