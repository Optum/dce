package main

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/data"
	"github.com/Optum/dce/pkg/model"
)

// GetAllAccounts - Returns all the accounts.
func GetAllAccounts(w http.ResponseWriter, r *http.Request) {
	// Fetch the accounts.
	dao := &data.Account{}
	if err := Services.Config.GetService(dao); err != nil {
		ErrorHandler(w, err)
		return
	}
	accounts, err := account.GetAccounts(dao)

	if err != nil {
		ErrorHandler(w, err)
		return
	}

	WriteAPIResponse(w, http.StatusOK, accounts)
}

// GetAccountByID - Returns the single account by ID
func GetAccountByID(w http.ResponseWriter, r *http.Request) {

	accountID := mux.Vars(r)["accountId"]

	dao := &data.Account{}
	if err := Services.Config.GetService(dao); err != nil {
		ErrorHandler(w, err)
		return
	}

	account, err := account.GetAccountByID(accountID, dao)

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

	dao := &data.Account{}
	if err := Services.Config.GetService(dao); err != nil {
		ErrorHandler(w, err)
		return
	}

	accounts, err := account.GetAccountsByStatus(status, dao)

	if err != nil {
		ErrorHandler(w, err)
		return
	}

	WriteAPIResponse(w, http.StatusOK, accounts)

}
