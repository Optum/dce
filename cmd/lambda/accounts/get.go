package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/errors"
)

// GetAllAccounts - Returns all the accounts.
func GetAllAccounts(w http.ResponseWriter, r *http.Request) {
	// Fetch the accounts.
	accounts, err := Dao.GetAccounts()

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to query database: %s", err)
		log.Print(errorMessage)
		api.WriteAPIErrorResponse(w, errors.NewInternalServer(errorMessage, nil))
		return
	}

	// Serialize them for the JSON response.
	accountResponses := []*response.AccountResponse{}

	for _, a := range accounts {
		acctRes := response.AccountResponse(*a)
		accountResponses = append(accountResponses, &acctRes)
	}

	json.NewEncoder(w).Encode(accountResponses)
}

// GetAccountByID - Returns the single account by ID
func GetAccountByID(w http.ResponseWriter, r *http.Request) {

	accountID := mux.Vars(r)["accountId"]
	account, err := Dao.GetAccount(accountID)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed List on Account Lease %s", accountID)
		log.Print(errorMessage)
		api.WriteAPIErrorResponse(w, errors.NewInternalServer(errorMessage, nil))
		return
	}

	if account == nil {
		api.WriteAPIErrorResponse(w, errors.NewNotFound("account", accountID))
		return
	}

	acctRes := response.AccountResponse(*account)

	json.NewEncoder(w).Encode(acctRes)
}

// GetAccountByStatus - Returns the accounts by status
func GetAccountByStatus(w http.ResponseWriter, r *http.Request) {
	// Fetch the accounts.
	accountStatus := r.FormValue("accountStatus")
	status, err := db.ParseAccountStatus(accountStatus)

	accounts, err := Dao.FindAccountsByStatus(status)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to query database: %s", err)
		log.Print(errorMessage)
		api.WriteAPIErrorResponse(w, errors.NewInternalServer(errorMessage, nil))
		return
	}

	if len(accounts) == 0 {
		api.WriteAPIErrorResponse(w, errors.NewNotFound("account", accountStatus))
		return
	}

	// Serialize them for the JSON response.
	accountResponses := []*response.AccountResponse{}

	for _, a := range accounts {
		acctRes := response.AccountResponse(*a)
		accountResponses = append(accountResponses, &acctRes)
	}

	json.NewEncoder(w).Encode(accountResponses)

}
