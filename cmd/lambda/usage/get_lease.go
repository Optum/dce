package main

import (
	"net/http"

	"github.com/Optum/dce/pkg/api"
	"github.com/gorilla/mux"
)

// GetLeaseUsageSummaryByLease lists Lease Usage information based the Lease ID
func GetLeaseUsageSummaryByLease(w http.ResponseWriter, r *http.Request) {
	leaseID := mux.Vars(r)["leaseID"]

	usg, err := Services.UsageService().GetLease(leaseID)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusOK, usg)

}
