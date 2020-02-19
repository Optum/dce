package main

import (
	"encoding/json"
	"fmt"
	"github.com/Optum/dce/pkg/api"
	"log"
	"net/http"
	"time"

	errors2 "github.com/Optum/dce/pkg/errors"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
)

type createLeaseRequest struct {
	PrincipalID              string                 `json:"principalId"`
	BudgetAmount             float64                `json:"budgetAmount"`
	BudgetCurrency           string                 `json:"budgetCurrency"`
	BudgetNotificationEmails []string               `json:"budgetNotificationEmails"`
	ExpiresOn                int64                  `json:"expiresOn"`
	Metadata                 map[string]interface{} `json:"metadata"`
}

// CreateLease - Creates the lease
func CreateLease(w http.ResponseWriter, r *http.Request) {
	c := leaseValidationContext{
		maxLeaseBudgetAmount:     maxLeaseBudgetAmount,
		maxLeasePeriod:           maxLeasePeriod,
		defaultLeaseLengthInDays: defaultLeaseLengthInDays,
		principalBudgetPeriod:    principalBudgetPeriod,
		principalBudgetAmount:    principalBudgetAmount,
	}

	// Extract the Body from the Request
	requestBody, isValid, validationErrorMessage, err := validateLeaseFromRequest(&c, r)

	if err != nil {
		response.WriteServerErrorWithResponse(w, err.Error())
		return
	}

	if !isValid {
		response.WriteRequestValidationError(w, validationErrorMessage)
		return
	}

	principalID := requestBody.PrincipalID

	// If user is not an admin, they can't create leases for other users
	if user.Role != api.AdminGroupName && principalID != user.Username {
		err = errors2.NewUnathorizedError(fmt.Sprintf("User [%s] with role: [%s] attempted to create a lease for: [%s], but was not authorized",
			user.Username, user.Role, principalID))
		api.WriteAPIErrorResponse(w, err)
		return
	}

	log.Printf("Creating lease for Principal %s", principalID)

	// Fail if the Principal already has an active lease
	principalLeases, err := dao.FindLeasesByPrincipal(requestBody.PrincipalID)
	if err != nil {
		log.Printf("Failed to list leases for principal %s: %s", requestBody.PrincipalID, err)
		response.WriteServerErrorWithResponse(w, "Internal server error")
		return
	}

	for _, lease := range principalLeases {
		if lease.LeaseStatus == db.Active {
			msg := fmt.Sprintf("Principal already has an active lease for account %s", lease.AccountID)
			response.WriteConflictError(w, msg)
			return
		}
	}

	// Get the First Ready Account
	// Exit if there's an error or no ready accounts
	account, err := dao.GetReadyAccount()
	if err != nil {
		log.Printf("Failed to Check Ready Accounts: %s", err)
		response.WriteServerErrorWithResponse(
			w,
			fmt.Sprintf("Failed to find a Ready Account: %s", err),
		)
		return
	} else if account == nil {
		errStr := "No Available accounts at this moment"
		log.Println(errStr)
		response.WriteServiceUnavailableError(w, errStr)
		return
	}
	log.Printf("Principal %s will be Leased to Account: %s\n", principalID,
		account.ID)

	// Create/Update lease record
	now := time.Now()
	lease, err := dao.UpsertLease(db.Lease{
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
		response.WriteServerError(w)
		return
	}

	// Mark the accOunt as Status=Leased
	_, err = dao.TransitionAccountStatus(account.ID, db.Ready, db.Leased)
	if err != nil {
		log.Printf("ERROR Failed to transition account %s to Leased for lease for %s. Attemping to deactivate lease...",
			lease.AccountID, lease.PrincipalID)
		// If setting the account status fails, attempt to deactivate the lease
		// before returning a 500 error
		_, err = dao.TransitionLeaseStatus(
			lease.AccountID, lease.PrincipalID,
			db.Active, db.Inactive, db.LeaseRolledBack,
		)
		if err != nil {
			log.Printf("Failed to deactivate lease on DB error for %s / %s: %s",
				lease.AccountID, lease.PrincipalID, err)
		}

		response.WriteServerError(w)
		return
	}

	// Publish Lease to the topic
	message, err := publishLease(snsSvc, lease, &leaseAddedTopicARN)
	if err != nil {
		log.Print(err.Error())

		// Attempt to rollback the lease
		_, err := dao.TransitionLeaseStatus(lease.AccountID, lease.PrincipalID, db.Active, db.Inactive, db.LeaseRolledBack)
		if err != nil {
			log.Printf("Failed to deactivate lease on SNS error for %s / %s: %s",
				lease.AccountID, lease.PrincipalID, err)
		} else {
			_, err := dao.TransitionAccountStatus(lease.AccountID, db.Leased, db.Ready)
			log.Printf("Failed to rollback account status on SNS error for %s / %s: %s",
				lease.AccountID, lease.PrincipalID, err)
		}

		response.WriteServerError(w)
		return
	}

	response.WriteAPIResponse(w, http.StatusCreated, *message)
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

// getBeginningOfCurrentBillingPeriod returns starts of the billing period based on budget period
func getBeginningOfCurrentBillingPeriod(input string) time.Time {
	currentTime := time.Now()
	if input == Weekly {

		for currentTime.Weekday() != time.Sunday { // iterate back to Sunday
			currentTime = currentTime.AddDate(0, 0, -1)
		}

		return time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC)
	}

	return time.Date(currentTime.Year(), currentTime.Month(), 1, 0, 0, 0, 0, time.UTC)
}
