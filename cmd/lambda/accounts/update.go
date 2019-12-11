package main

import (
	"encoding/json"
	"net/http"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/accountmanager"
	"github.com/Optum/dce/pkg/data"
	"github.com/Optum/dce/pkg/model"
	"github.com/gorilla/mux"
)

// UpdateAccountByID updates an accounts information based on ID
func UpdateAccountByID(w http.ResponseWriter, r *http.Request) {
	accountID := mux.Vars(r)["accountId"]

	// Deserialize the request JSON as an request object
	d := model.Account{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&d)
	if err != nil {
		ErrorHandler(w, err)
		return
	}

	dao := &data.Account{}
	if err := Services.Config.GetService(dao); err != nil {
		ErrorHandler(w, err)
		return
	}

	am := &accountmanager.AccountManager{}
	if err := Services.Config.GetService(am); err != nil {
		ErrorHandler(w, err)
		return
	}

	account, err := account.GetAccountByID(accountID, dao)
	err = account.Update(d, dao, am)

	if err != nil {
		ErrorHandler(w, err)
		return
	}

	WriteAPIResponse(w, http.StatusOK, account)
}
