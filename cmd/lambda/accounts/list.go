package main

import (
	"encoding/json"
	"fmt"
	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/errors"
	"log"
	"net/http"

	"github.com/Optum/dce/pkg/api/response"
)

// GetAllAccounts - Returns all the accounts.
func GetAllAccounts(w http.ResponseWriter, r *http.Request) {
	// Fetch the accounts.
	accounts, err := Dao.GetAccounts()

	if err != nil {
		errorMessage := fmt.Sprintf("Failed to query database: %s", err)
		log.Print(errorMessage)
		api.WriteAPIErrorResponse(w, errors.NewInternalServer(errorMessage, nil))
		return
	}

	// Serialize them for the JSON response.
	accountResponses := []*response.AccountResponse{}

	for _, a := range accounts {
		acctRes := response.AccountResponse(*a)
		accountResponses = append(accountResponses, &acctRes)
	}

	_ = json.NewEncoder(w).Encode(accountResponses)
}
