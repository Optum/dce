package main

import (
	"fmt"
	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/lease"
	"github.com/gorilla/schema"
	"net/http"
)

// GetLeases - Returns leases
func GetLeases(w http.ResponseWriter, r *http.Request) {
	// Fetch the leases.

	var decoder = schema.NewDecoder()

	query := &lease.Lease{}
	err := decoder.Decode(query, r.URL.Query())
	if err != nil {
		response.WriteRequestValidationError(w, fmt.Sprintf("Error parsing query params"))
		return
	}

	leases, err := Services.LeaseService().List(query)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	if query.NextAccountID != nil && query.NextPrincipalID != nil {
		nextURL, err := api.BuildNextURL(baseRequest, query)
		if err != nil {
			api.WriteAPIErrorResponse(w, err)
			return
		}
		w.Header().Add("Link", fmt.Sprintf("<%s>; rel=\"next\"", nextURL.String()))
	}
	api.WriteAPIResponse(w, http.StatusOK, leases)

}
