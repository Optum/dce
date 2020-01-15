package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/Optum/dce/pkg/db"

	"github.com/Optum/dce/pkg/api/response"
)

// GetLeases - Gets all of the leases
func GetLeases(w http.ResponseWriter, r *http.Request) {

	// This has become a "fall-through" method for any of the URL combinations that
	// don't match the explicit routes, so we parse input here to get all of the
	// query string values that are supplied on the URL
	getLeasesInput, err := parseGetLeasesInput(r)

	if err != nil {
		log.Print(err)
		response.WriteRequestValidationError(w, fmt.Sprintf("Error parsing query params"))
		return
	}

	result, err := dao.GetLeases(getLeasesInput)

	if err != nil {
		log.Print(err)
		response.WriteServerError(w)
		return
	}

	// Convert DB Lease model to API Response model
	leaseResponseItems := []response.LeaseResponse{}
	for _, lease := range result.Results {
		leaseResponseItems = append(leaseResponseItems, response.LeaseResponse(*lease))
	}

	// If the DB result has next keys, then the URL to retrieve the next page is put into the Link header.
	if len(result.NextKeys) > 0 {
		nextURL := response.BuildNextURL(r, result.NextKeys, baseRequest)
		w.Header().Add("Link", fmt.Sprintf("<%s>; rel=\"next\"", nextURL.String()))
	}

	err = json.NewEncoder(w).Encode(leaseResponseItems)
	if err != nil {
		log.Print(err)
		response.WriteServerError(w)
	}
}

// parseGetLeasesInput creates a GetLeasesInput from the query parameters
func parseGetLeasesInput(r *http.Request) (db.GetLeasesInput, error) {
	query := db.GetLeasesInput{
		StartKeys: make(map[string]string),
	}

	statusValue := r.FormValue(StatusParam)
	if len(statusValue) > 0 {
		status, err := db.ParseLeaseStatus(statusValue)
		if err != nil {
			return query, err
		}
		if len(status) > 0 {
			query.Status = status
		}
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
