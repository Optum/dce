package main

import (
	"encoding/json"
	"net/http"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/accountmanager/accountmanageriface"
	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/data/dataiface"
	"github.com/Optum/dce/pkg/errors"
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
		api.WriteAPIErrorResponse(w,
			errors.NewBadRequest("invalid request parameters"))
		return
	}

	var dataSvc dataiface.AccountData
	if err := Services.Config.GetService(&dataSvc); err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	var amSvc accountmanageriface.AccountManagerAPI
	if err := Services.Config.GetService(&amSvc); err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	a, err := account.GetAccountByID(accountID, dataSvc, dataSvc)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}
	err = a.Update(d, amSvc)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusOK, a)
}
