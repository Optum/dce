package main

import (
	"encoding/json"
	"net/http"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
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

	// Mark the account as Status=Leased
	availableAccount.Status = account.StatusLeased.StatusPtr()
	leasedAccount, err := Services.AccountService().Update(*availableAccount.ID, &availableAccount)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	// Create lease
	newLease.AccountID = leasedAccount.ID
	lease, err := Services.LeaseService().Create(newLease)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusCreated, lease)
}
