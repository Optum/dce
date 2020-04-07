package main

import (
	"encoding/json"
	"github.com/Optum/dce/pkg/api"
	"github.com/aws/aws-sdk-go/service/sfn"
	"log"
	"net/http"
	"time"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
)

const (
	Weekly = "WEEKLY"
)

// CreateLease - Function to validate the lease request and create lease
func CreateLease(w http.ResponseWriter, r *http.Request) {
	// Deserialize the request JSON as an request object
	newLease := &lease.Lease{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(newLease)

	if err != nil {
		api.WriteAPIErrorResponse(w,
			errors.NewBadRequest("invalid request parameters"))
		return
	}

	// if principalId is missing, then throw an error
	if newLease.PrincipalID == nil {
		api.WriteAPIErrorResponse(w,
			errors.NewBadRequest("invalid request parameters: missing principalId"))
		return
	}

	// If user is not an admin, they can't create leases for other users
	user := r.Context().Value(api.User{}).(*api.User)
	err = user.Authorize(*newLease.PrincipalID)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	// Get the First available Ready Account
	query := &account.Account{
		Status: account.StatusReady.StatusPtr(),
	}

	accounts, err := Services.AccountService().List(query)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}
	if accounts == nil || len(*accounts) == 0 {
		api.WriteAPIErrorResponse(w,
			errors.NewInternalServer("No Available accounts at this moment", nil))
		return
	}
	availableAccount := (*accounts)[0]

	// Check if an inactive lease already exists with same principal id and account id
	// if an inactive lease exists, then get the lastModifiedOn value from it
	queryLeases := &lease.Lease{
		AccountID:   availableAccount.ID,
		PrincipalID: newLease.PrincipalID,
		Status:      lease.StatusInactive.StatusPtr(),
	}
	foundLeases, err := Services.LeaseService().List(queryLeases)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	// Since we are using primary key to query, the number of leases that match the query should be one
	if foundLeases != nil && len(*foundLeases) == 1 {
		newLease.LastModifiedOn = (*foundLeases)[0].LastModifiedOn
		newLease.CreatedOn = (*foundLeases)[0].CreatedOn
	} else {
		newLease.LastModifiedOn = nil
	}

	// Create lease
	newLease.AccountID = availableAccount.ID
	leaseCreated, err := Services.LeaseService().Create(newLease)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	// Mark the account as Status=Leased
	availableAccount.Status = account.StatusLeased.StatusPtr()
	_, err = Services.AccountService().Update(*availableAccount.ID, &availableAccount)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	// Start the step function to track usage for this lease
	sfnSvc := Services.StepFunctions()
	sfnInput := lease.Lease{
		AccountID:                newLease.AccountID,
		PrincipalID:              newLease.PrincipalID,
		ID:                       newLease.ID,
		Status:                   newLease.Status.StatusPtr(),
		StatusReason:             newLease.StatusReason.StatusReasonPtr(),
		CreatedOn:                newLease.CreatedOn,
		LastModifiedOn:           newLease.LastModifiedOn,
		BudgetAmount:             newLease.BudgetAmount,
		BudgetCurrency:           newLease.BudgetCurrency,
		BudgetNotificationEmails: newLease.BudgetNotificationEmails,
		StatusModifiedOn:         newLease.StatusModifiedOn,
		ExpiresOn:                newLease.ExpiresOn,
	}
	sfnInputBytes, err := json.Marshal(sfnInput)
	if err != nil {
		log.Printf("Failed to retrieve step functions service %s", err)
	}
	sfnInputString := string(sfnInputBytes)
	_, err = sfnSvc.StartExecution(&sfn.StartExecutionInput{
		Input:           &sfnInputString,
		StateMachineArn: &Settings.UsageStepFunctionArn,
	})
	if err != nil {
		log.Printf("ERROR: Failed to start step function execution %s", err)
		api.WriteAPIErrorResponse(w,
			errors.NewInternalServer("Lease creation failed", nil))
		return
	}

	api.WriteAPIResponse(w, http.StatusCreated, leaseCreated)
}

// getStartOfPrincipalBudgetPeriod returns starts of the billing period based on budget period
func getStartOfPrincipalBudgetPeriod(input string) time.Time { //nolint
	currentTime := time.Now()
	if input == Weekly {

		for currentTime.Weekday() != time.Sunday { // iterate back to Sunday
			currentTime = currentTime.AddDate(0, 0, -1)
		}

		return time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC)
	}

	return time.Date(currentTime.Year(), currentTime.Month(), 1, 0, 0, 0, 0, time.UTC)
}
