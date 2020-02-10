package main

import (
	"net/http"

	"github.com/Optum/dce/pkg/api"
	"github.com/gorilla/mux"
)

// DeleteAccount - Deletes the account
func DeleteAccount(w http.ResponseWriter, r *http.Request) {

	accountID := mux.Vars(r)["accountId"]

	acct, err := Services.AccountService().Get(accountID)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	err = Services.AccountService().Delete(acct)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusNoContent, nil)
}
