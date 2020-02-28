package main

import (
	"encoding/json"
	"github.com/Optum/dce/pkg/api"
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
	if accounts != nil && len(*accounts) == 0 {
		api.WriteAPIErrorResponse(w,
			errors.NewInternalServer("No Available accounts at this moment", nil))
		return
	}
	availableAccount := (*accounts)[0]

	// Get user principal's current spend
	usageStartTime := getBeginningOfCurrentBillingPeriod(Settings.PrincipalBudgetPeriod)
	usageRecords, err := usageSvc.GetUsageByPrincipal(usageStartTime, *newLease.PrincipalID)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	// Group by PrincipalID to get sum of total spent for current billing period
	spent := 0.0
	for _, usageItem := range usageRecords {
		spent = spent + *usageItem.CostAmount
	}

	// Create lease
	newLease.AccountID = availableAccount.ID
	leaseCreated, err := Services.LeaseService().Create(newLease, spent)
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

	api.WriteAPIResponse(w, http.StatusCreated, leaseCreated)
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
