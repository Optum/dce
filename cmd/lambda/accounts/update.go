package main

import (
	"encoding/json"
	"net/http"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/errors"
	validation "github.com/go-ozzo/ozzo-validation"
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

	err = validation.ValidateStruct(newAccount,
		// ID has to be empty
		validation.Field(&newAccount.ID, validation.NilOrNotEmpty, validation.In(accountID)),
		validation.Field(&newAccount.LastModifiedOn, validation.By(isNil)),
		validation.Field(&newAccount.Status, validation.By(isNil)),
		validation.Field(&newAccount.CreatedOn, validation.By(isNil)),
		validation.Field(&newAccount.PrincipalRoleArn, validation.By(isNil)),
		validation.Field(&newAccount.PrincipalPolicyHash, validation.By(isNil)),
	)
	if err != nil {
		api.WriteAPIErrorResponse(w,
			errors.NewValidation("account", err))
		return
	}

	account, err := Services.AccountService().Update(accountID, newAccount)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusOK, account)
}
