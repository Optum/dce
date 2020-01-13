package main

import (
	"fmt"
	"net/http"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/data/dataiface"
	"github.com/Optum/dce/pkg/model"
	"github.com/gorilla/schema"
)

// GetAccounts - Returns accounts
func GetAccounts(w http.ResponseWriter, r *http.Request) {
	// Fetch the accounts.

	var decoder = schema.NewDecoder()

	q := model.Account{}
	err := decoder.Decode(&q, r.URL.Query())
	if err != nil {
		response.WriteRequestValidationError(w, fmt.Sprintf("Error parsing query params"))
		return
	}

	var dataSvc dataiface.AccountData

	if err := Services.Config.GetService(&dataSvc); err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	accounts, err := account.GetAccounts(&q, dataSvc)

	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	if q.NextID != nil {
		nextURL, err := api.BuildNextURL(baseRequest, q)
		if err != nil {
			api.WriteAPIErrorResponse(w, err)
			return
		}
		w.Header().Add("Link", fmt.Sprintf("<%s>; rel=\"next\"", nextURL.String()))
	}
	api.WriteAPIResponse(w, http.StatusOK, accounts)

}
