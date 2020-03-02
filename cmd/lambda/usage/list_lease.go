package main

import (
	"fmt"
	"net/http"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/usage"
	"github.com/gorilla/schema"
)

// ListLeaseUsageSummary - Returns leases
func ListLeaseUsageSummary(w http.ResponseWriter, r *http.Request) {
	// Fetch the leases.

	var decoder = schema.NewDecoder()

	query := &usage.Lease{}
	err := decoder.Decode(query, r.URL.Query())
	if err != nil {
		response.WriteRequestValidationError(w, fmt.Sprintf("Error parsing query params"))
		return
	}

	listLeases(query, w, r)
}

func listLeases(query *usage.Lease, w http.ResponseWriter, r *http.Request) {

	usgs, err := Services.UsageService().ListLease(query)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	if query.NextLeaseID != nil && query.NextDate != nil {
		nextURL, err := api.BuildNextURL(baseRequest, query)
		if err != nil {
			api.WriteAPIErrorResponse(w, err)
			return
		}
		w.Header().Add("Link", fmt.Sprintf("<%s>; rel=\"next\"", nextURL.String()))
	}
	api.WriteAPIResponse(w, http.StatusOK, usgs)
}
