package main

import (
	"fmt"
	"net/http"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/usage"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

// GetLeaseUsageSummaryByLease lists Lease Usage information based the Lease ID
func GetLeaseUsageSummaryByLease(w http.ResponseWriter, r *http.Request) {
	leaseID := mux.Vars(r)["leaseID"]

	query := &usage.Lease{
		LeaseID: &leaseID,
	}

	listLeases(query, w, r)

}

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
	}

	if query.NextLeaseID != nil && query.NextDate != nil {
		nextURL, err := api.BuildNextURL(baseRequest, query)
		if err != nil {
			api.WriteAPIErrorResponse(w, err)
		}
		w.Header().Add("Link", fmt.Sprintf("<%s>; rel=\"next\"", nextURL.String()))
	}
	api.WriteAPIResponse(w, http.StatusOK, usgs)
}
