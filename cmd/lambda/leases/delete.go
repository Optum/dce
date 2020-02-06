package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/lease"
	"github.com/gorilla/mux"
)

// DeleteLeaseByID - Deletes the given lease by Lease ID
func DeleteLeaseByID(w http.ResponseWriter, r *http.Request) {

	leaseID := mux.Vars(r)["leaseID"]

	lease, err := Services.Config.LeaseService().Delete(leaseID)

	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusOK, lease)
}

// DeleteLease - Deletes the given lease
func DeleteLease(w http.ResponseWriter, r *http.Request) {

	// Deserialize the request JSON as an request object
	queryLease := &lease.Lease{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(queryLease)
	if err != nil {
		api.WriteAPIErrorResponse(w,
			errors.NewBadRequest("invalid request parameters"))
		return
	}

	leases, err := Services.Config.LeaseService().List(queryLease)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	if len(*leases) == 0 {
		response.WriteRequestValidationError(w, fmt.Sprintf("No leases found for Principal %q and Account ID %q", *queryLease.PrincipalID, *queryLease.AccountID))
		return
	}

	if len(*leases) > 1 {
		response.WriteRequestValidationError(w, fmt.Sprintf("Found more than one lease"))
		return
	}
	leaseID := (*leases)[0].ID
	lease, err := Services.Config.LeaseService().Delete(*leaseID)

	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusOK, lease)
}
