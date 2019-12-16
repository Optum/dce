package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Optum/dce/pkg/usage"
	"github.com/Optum/dce/pkg/api/response"
)

// GetUsage - Gets all of the usage
func GetUsage(w http.ResponseWriter, r *http.Request) {

	// This has become a "fall-through" method for any of the URL combinations that
	// don't match the explicit routes, so we parse input here to get all of the
	// query string values that are supplied on the URL
	getUsageInput, err := parseGetUsageInput(r)

	if err != nil {
		response.WriteRequestValidationError(w, fmt.Sprintf("Error parsing query params"))
		return
	}

	result, err := UsageSvc.GetUsage(getUsageInput)

	if err != nil {
		response.WriteServerErrorWithResponse(w, fmt.Sprintf("Error querying usage: %s", err))
		return
	}

	// Serialize them for the JSON response.
	usageResponseItems := []response.UsageResponse{}
	for _, usageItem := range result.Results {
		usageResponseItems = append(usageResponseItems, response.UsageResponse(*usageItem))
	}

	// If the DB result has next keys, then the URL to retrieve the next page is put into the Link header.
	if len(result.NextKeys) > 0 {
		nextURL := buildNextURL(r, result.NextKeys)
		w.Header().Add("Link", fmt.Sprintf("<%s>; rel=\"next\"", nextURL))
	}

	json.NewEncoder(w).Encode(usageResponseItems)
}

// parseGetUsageInput creates a GetUsageInput from the query parameters
func parseGetUsageInput(r *http.Request) (usage.GetUsageInput, error) {
	query := usage.GetUsageInput{
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

	i, err := strconv.ParseInt(r.FormValue(StartDateParam), 10, 64)
	if err != nil {
		return query, err
	}
	startDate := time.Unix(i, 0)
	if startDate != *new(time.Time) {
		query.StartDate = startDate
	}

	principalID := r.FormValue(PrincipalIDParam)
	if len(principalID) > 0 {
		query.PrincipalID = principalID
	}

	accountID := r.FormValue(AccountIDParam)
	if len(accountID) > 0 {
		query.AccountID = accountID
	}

	nextStartDate := r.FormValue(NextStartDateParam)
	if len(nextStartDate) > 0 {
		query.StartKeys["StartDate"] = nextStartDate
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
	return fmt.Sprintf("%s?%s", r.URL.EscapedPath(), queryString)
}
