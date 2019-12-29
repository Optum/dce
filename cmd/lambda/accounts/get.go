package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/db"
)

// GetAllAccounts - Returns all the accounts.
func GetAllAccounts(w http.ResponseWriter, r *http.Request) {
	// Fetch the accounts.
	var dao db.DBer
	if err := services.Config.GetService(&dao); err != nil {
		response.WriteServerErrorWithResponse(w, "Could not create data service")
		return
	}
	accounts, err := dao.GetAccounts()

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to query database: %s", err)
		log.Print(errorMessage)
		response.WriteServerErrorWithResponse(w, errorMessage)
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
	var dao db.DBer
	if err := services.Config.GetService(&dao); err != nil {
		response.WriteServerErrorWithResponse(w, "Could not create data service")
		return
	}
	account, err := dao.GetAccount(accountID)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed List on Account Lease %s", accountID)
		log.Print(errorMessage)
		response.WriteServerErrorWithResponse(w, errorMessage)
		return
	}

	if account == nil {
		response.WriteNotFoundError(w)
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

	var dao db.DBer
	if err := services.Config.GetService(&dao); err != nil {
		response.WriteServerErrorWithResponse(w, "Could not create data service")
		return
	}
	accounts, err := dao.FindAccountsByStatus(status)

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to query database: %s", err)
		log.Print(errorMessage)
		response.WriteServerErrorWithResponse(w, errorMessage)
	}

	if len(accounts) == 0 {
		response.WriteNotFoundError(w)
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
