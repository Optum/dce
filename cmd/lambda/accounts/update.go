package main

import (
	"encoding/json"
	"net/http"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/errors"
	"github.com/gorilla/mux"
)

// UpdateAccountByID updates an accounts information based on ID
func UpdateAccountByID(w http.ResponseWriter, r *http.Request) {
	accountID := mux.Vars(r)["accountId"]

	// Deserialize the request JSON as an request object
	newAccount := &account.Account{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(newAccount)
	if err != nil {
		api.WriteAPIErrorResponse(w,
			errors.NewBadRequest("invalid request parameters"))
		return
	}

	account, err := Services.Config.AccountSvc().Update(accountID, newAccount)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusOK, account)
}
