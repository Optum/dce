package main

import (
	"encoding/json"
	"net/http"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/errors"
)

// CreateAccount - Function to validate the account request to add into the pool and
// publish the account creation to its respective client
func CreateAccount(w http.ResponseWriter, r *http.Request) {
	// Deserialize the request JSON as an request object
	newAccount := &account.Account{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(newAccount)
	if err != nil {
		api.WriteAPIErrorResponse(w,
			errors.NewBadRequest("invalid request parameters"))
		return
	}

	account, err := Services.AccountService().Create(newAccount)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusCreated, account)
}
