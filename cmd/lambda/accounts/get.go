package main

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/model"
)

// GetAllAccounts - Returns all the accounts.
func GetAllAccounts(w http.ResponseWriter, r *http.Request) {
	// Fetch the accounts.
	accounts, err := account.GetAccounts(DataSvc)

	if err != nil {
		ErrorHandler(w, err)
		return
	}

	WriteAPIResponse(w, http.StatusOK, accounts)
}

// GetAccountByID - Returns the single account by ID
func GetAccountByID(w http.ResponseWriter, r *http.Request) {

	accountID := mux.Vars(r)["accountId"]
	account, err := account.GetAccountByID(accountID, DataSvc)

	if err != nil {
		ErrorHandler(w, err)
		return
	}

	WriteAPIResponse(w, http.StatusOK, account)
}

// GetAccountByStatus - Returns the accounts by status
func GetAccountByStatus(w http.ResponseWriter, r *http.Request) {
	// Fetch the accounts.
	accountStatus := r.FormValue("accountStatus")
	status := model.AccountStatus(accountStatus)

	accounts, err := account.GetAccountsByStatus(status, DataSvc)

	if err != nil {
		ErrorHandler(w, err)
		return
	}

	WriteAPIResponse(w, http.StatusOK, accounts)

}
