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

	// Get the First available Ready Account
	query := &account.Account{
		Status: account.StatusReady.StatusPtr(),
	}

	accounts, err := Services.AccountService().List(query)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}
	if len(*accounts) == 0 {
		api.WriteAPIErrorResponse(w,
			errors.NewInternalServer("No Available accounts at this moment", nil))
		return
	}
	availableAccount := (*accounts)[0]

	// If user is not an admin, they can't create leases for other users
	user := r.Context().Value(api.User{}).(*api.User)
	err = user.Authorize(*newLease.PrincipalID)

	// Mark the account as Status=Leased
	availableAccount.Status = account.StatusLeased.StatusPtr()
	leasedAccount, err := Services.AccountService().Update(*availableAccount.ID, &availableAccount)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	// Get user principal's current spend
	principalBudgetPeriod, err := Services.Config.GetStringVal("PrincipalBudgetPeriod")
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	usageStartTime := getBeginningOfCurrentBillingPeriod(principalBudgetPeriod).Unix()
	usageRecords, err := Services.UsageService().Get(usageStartTime, *newLease.PrincipalID)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}
	// Group by PrincipalID to get sum of total spent for current billing period
	spent := 0.0
	if usageRecords != nil {
		for _, usageItem := range *usageRecords {
			spent = spent + *usageItem.CostAmount
		}
	}

	// Create lease
	newLease.AccountID = leasedAccount.ID
	lease, err := Services.LeaseService().Create(newLease, spent)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusCreated, lease)
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
