package main

import (
	"fmt"
	"net/http"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/api/response"
	"github.com/gorilla/schema"
)

// GetAccounts - Returns accounts
func GetAccounts(w http.ResponseWriter, r *http.Request) {
	// Fetch the accounts.

	var decoder = schema.NewDecoder()

	query := &account.Account{}
	err := decoder.Decode(query, r.URL.Query())
	if err != nil {
		response.WriteRequestValidationError(w, fmt.Sprintf("Error parsing query params"))
		return
	}

	accounts, err := Services.Config.AccountService().List(query)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	if query.NextID != nil {
		nextURL, err := api.BuildNextURL(baseRequest, query)
		if err != nil {
			api.WriteAPIErrorResponse(w, err)
			return
		}
		w.Header().Add("Link", fmt.Sprintf("<%s>; rel=\"next\"", nextURL.String()))
	}
	api.WriteAPIResponse(w, http.StatusOK, accounts)

}
