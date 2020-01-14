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
	var dataSvc dataiface.AccountData

	if err := Services.GetService(&dataSvc); err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	account, err := account.GetAccountByID(accountID, dataSvc, nil)

	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusOK, account)
}
