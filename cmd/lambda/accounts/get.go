package main

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/data/dataiface"
)

// GetAccountByID - Returns the single account by ID
func GetAccountByID(w http.ResponseWriter, r *http.Request) {

	accountID := mux.Vars(r)["accountId"]
	var dao dataiface.AccountData

	if err := Services.Config.GetService(&dao); err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	account, err := account.GetAccountByID(accountID, dao, nil)

	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusOK, account)
}
