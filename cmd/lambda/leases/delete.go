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
	_lease, err := Services.LeaseService().Get(leaseID)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	//If user is not an admin, they can't delete leases for other users
	if user.Role != api.AdminGroupName && *_lease.PrincipalID != user.Username {
		m := fmt.Sprintf("User [%s] with role: [%s] attempted to delete a lease for: [%s], but was not authorized",
			user.Username, user.Role, *_lease.PrincipalID)
		api.WriteAPIErrorResponse(w,
			errors.NewUnathorizedError(m))
		return
	}

	deletedLease, err := Services.LeaseService().Delete(leaseID)

	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusOK, deletedLease)
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

	if queryLease.AccountID == nil {
		api.WriteAPIErrorResponse(w,
			errors.NewBadRequest("invalid request parameters: missing AccountID"))
		return
	}

	if queryLease.PrincipalID == nil {
		api.WriteAPIErrorResponse(w,
			errors.NewBadRequest("invalid request parameters: missing PrincipalID"))
		return
	}

	// If user is not an admin, they can't delete leases for other users
	if user.Role != api.AdminGroupName && *queryLease.PrincipalID != user.Username {
		m := fmt.Sprintf("User [%s] with role: [%s] attempted to delete a lease for: [%s], but was not authorized",
			user.Username, user.Role, *queryLease.PrincipalID)
		api.WriteAPIErrorResponse(w,
			errors.NewUnathorizedError(m))
		return
	}

	leases, err := Services.LeaseService().List(queryLease)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	if len(*leases) == 0 {
		api.WriteAPIErrorResponse(w,
			errors.NewNotFound("lease", fmt.Sprintf("with Principal ID %s and Account ID %s", *queryLease.PrincipalID, *queryLease.AccountID)))
		return
	}

	if len(*leases) > 1 {
		response.WriteRequestValidationError(w, fmt.Sprintf("Found more than one lease"))
		return
	}

	leaseID := (*leases)[0].ID
	deletedLease, err := Services.LeaseService().Delete(*leaseID)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusOK, deletedLease)
}
