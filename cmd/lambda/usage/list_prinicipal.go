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

// ListPrincipalUsageByPrincipal lists Principal Usage information based on the Principal ID
func ListPrincipalUsageByPrincipal(w http.ResponseWriter, r *http.Request) {
	principalID := mux.Vars(r)["principalID"]

	query := &usage.Principal{
		PrincipalID: &principalID,
	}

	listPrincipal(query, w, r)

}

// ListPrincipalUsage - Returns leases
func ListPrincipalUsage(w http.ResponseWriter, r *http.Request) {
	// Fetch the leases.

	var decoder = schema.NewDecoder()

	query := &usage.Principal{}
	err := decoder.Decode(query, r.URL.Query())
	if err != nil {
		response.WriteRequestValidationError(w, "Error parsing query params")
		return
	}

	listPrincipal(query, w, r)

}

func listPrincipal(query *usage.Principal, w http.ResponseWriter, r *http.Request) {

	usgs, err := Services.UsageService().ListPrincipal(query)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	if query.NextPrincipalID != nil && query.NextDate != nil {
		nextURL, err := api.BuildNextURL(baseRequest, query)
		if err != nil {
			api.WriteAPIErrorResponse(w, err)
			return
		}
		w.Header().Add("Link", fmt.Sprintf("<%s>; rel=\"next\"", nextURL.String()))
	}
	api.WriteAPIResponse(w, http.StatusOK, usgs)
}
