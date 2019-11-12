package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Optum/dce/pkg/db"

	"github.com/Optum/dce/pkg/api/response"
)

// GetLeases - Gets all of the leases
func GetLeases(w http.ResponseWriter, r *http.Request) {

	getLeasesInput, err := parseGetLeasesInput(r)

	if err != nil {
		response.WriteRequestValidationError(w, fmt.Sprintf("Error parsing query params"))
		return
	}

	result, err := Dao.GetLeases(getLeasesInput)

	if err != nil {
		response.WriteServerErrorWithResponse(w, fmt.Sprintf("Error querying leases: %s", err))
		return
	}

	// Convert DB Lease model to API Response model
	leaseResponseItems := []response.LeaseResponse{}
	for _, lease := range result.Results {
		leaseResponseItems = append(leaseResponseItems, response.LeaseResponse(*lease))
	}

	if err != nil {
		response.WriteServerErrorWithResponse(w, fmt.Sprintf("Error serializing response: %s", err))
		return
	}

	// If the DB result has next keys, then the URL to retrieve the next page is put into the Link header.
	if len(result.NextKeys) > 0 {
		nextURL := buildNextURL(r, result.NextKeys)
		w.Header().Add("Link", fmt.Sprintf("<%s>; rel=\"next\"", nextURL))
	}

	json.NewEncoder(w).Encode(leaseResponseItems)
}

// parseGetLeasesInput creates a GetLeasesInput from the query parameters
func parseGetLeasesInput(r *http.Request) (db.GetLeasesInput, error) {
	query := db.GetLeasesInput{
		StartKeys: make(map[string]string),
	}

	status := r.FormValue(StatusParam)

	if len(status) > 0 {
		query.Status = status
	}

	limit := r.FormValue(LimitParam)
	if len(limit) > 0 {
		limInt, err := strconv.ParseInt(limit, 10, 64)
		query.Limit = limInt
		if err != nil {
			return query, err
		}
	}

	principalID := r.FormValue(PrincipalIDParam)
	if len(principalID) > 0 {
		query.PrincipalID = principalID
	}

	accountID := r.FormValue(AccountIDParam)
	if len(accountID) > 0 {
		query.AccountID = accountID
	}

	nextAccountID := r.FormValue(NextAccountIDParam)
	if len(nextAccountID) > 0 {
		query.StartKeys["AccountId"] = nextAccountID
	}

	nextPrincipalID := r.FormValue(NextPrincipalIDParam)
	if len(nextPrincipalID) > 0 {
		query.StartKeys["PrincipalId"] = nextPrincipalID
	}

	return query, nil
}

// buildNextURL merges the next parameters into the request parameters and returns an API URL.
func buildNextURL(r *http.Request, nextParams map[string]string) string {
	responseParams := make(map[string]string)
	responseQueryStrings := make([]string, 0)
	base := buildBaseURL(r)

	for k, v := range r.URL.Query() {
		responseParams[k] = v[0]
	}

	for k, v := range nextParams {
		responseParams[fmt.Sprintf("next%s", k)] = v
	}

	for k, v := range responseParams {
		responseQueryStrings = append(responseQueryStrings, fmt.Sprintf("%s=%s", k, v))
	}

	queryString := strings.Join(responseQueryStrings, "&")
	return fmt.Sprintf("%s%s?%s", base, r.URL.EscapedPath(), queryString)
}
