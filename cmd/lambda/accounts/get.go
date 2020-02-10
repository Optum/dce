package main

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Optum/dce/pkg/api"
)

// GetAccountByID - Returns the single account by ID
func GetAccountByID(w http.ResponseWriter, r *http.Request) {

	accountID := mux.Vars(r)["accountId"]

	account, err := Services.AccountService().Get(accountID)

	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusOK, account)
}
