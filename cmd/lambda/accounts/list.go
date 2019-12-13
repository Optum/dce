package main

import (
	"net/http"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/data"
	"github.com/Optum/dce/pkg/model"
)

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

// GetAccounts - Returns all the accounts.
func GetAccounts(w http.ResponseWriter, r *http.Request) {
	// Fetch the accounts.
	dao := &data.Account{}
	if err := Services.Config.GetService(dao); err != nil {
		ErrorHandler(w, err)
		return
	}

	query := &model.Account{}
	err := api.GetStructFromQuery(query, r.URL.Query())
	if err != nil {
		ErrorHandler(w, err)
		return
	}
	accounts, err := account.GetAccounts(query, dao)

	if err != nil {
		ErrorHandler(w, err)
		return
	}

	WriteAPIResponse(w, http.StatusOK, accounts)
}
