package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/Optum/dce/pkg/db"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/errors"

	"github.com/Optum/dce/pkg/api/response"
)

// GetAllAccounts - Returns all the accounts.
func GetAllAccounts(w http.ResponseWriter, r *http.Request) {
	// Fetch the accounts.

	getAccountsInput, err := parseGetAccountsInput(r)

	if err != nil {
		log.Print(err)
		response.WriteRequestValidationError(w, fmt.Sprintf("Error parsing query params"))
		return
	}

	result, err := Dao.GetAccounts(getAccountsInput)

	if err != nil {
		log.Print(err)
		response.WriteServerError(w)
		return
	}

	// Serialize them for the JSON response.
	accountResponses := []*response.AccountResponse{}

	for _, a := range result.Results {
		acctRes := response.AccountResponse(*a)
		accountResponses = append(accountResponses, &acctRes)
	}

	// If the DB result has next keys, then the URL to retrieve the next page is put into the Link header.
	if len(result.NextKeys) > 0 {
		nextURL := response.BuildNextURL(r, result.NextKeys, baseRequest)
		w.Header().Add("Link", fmt.Sprintf("<%s>; rel=\"next\"", nextURL.String()))
	}

	err = json.NewEncoder(w).Encode(accountResponses)
	if err != nil {
		log.Print(err)
		response.WriteServerError(w)
	}
}

// parseGetAccountsInput creates a GetAccountsInput from the query parameters
func parseGetAccountsInput(r *http.Request) (db.GetAccountsInput, error) {
	query := db.GetAccountsInput{
		StartKeys: make(map[string]string),
	}

	limit := r.FormValue(LimitParam)
	if len(limit) > 0 {
		limInt, err := strconv.ParseInt(limit, 10, 64)
		query.Limit = limInt
		if err != nil {
			return query, err
		}
	}

	statusValue := r.FormValue(StatusParam)
	if len(statusValue) > 0 {
		status, err := db.ParseAccountStatus(statusValue)
		if err != nil {
			return query, err
		}
		if len(status) > 0 {
			query.Status = status
		}
	}

	accountID := r.FormValue(AccountIDParam)
	if len(accountID) > 0 {
		query.AccountID = accountID
	}

	nextAccountID := r.FormValue(NextAccountIDParam)
	if len(nextAccountID) > 0 {
		query.StartKeys["Id"] = nextAccountID
	}

	return query, nil
}

// GetAccountByStatus - Returns the accounts by status
func GetAccountByStatus(w http.ResponseWriter, r *http.Request) {
	// Fetch the accounts.
	accountStatus := r.FormValue("accountStatus")
	status, err := db.ParseAccountStatus(accountStatus)
	if err != nil {
		log.Print(err)
		api.WriteAPIErrorResponse(w, errors.NewValidation(err.Error(), nil))
		return
	}

	accounts, err := Dao.FindAccountsByStatus(status)
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to query database: %s", err)
		log.Print(errorMessage)
		api.WriteAPIErrorResponse(w, errors.NewInternalServer(errorMessage, nil))
		return
	}

	if len(accounts) == 0 {
		api.WriteAPIErrorResponse(w, errors.NewNotFound("account", accountStatus))
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
