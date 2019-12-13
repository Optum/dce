package main

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/data"
)

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
