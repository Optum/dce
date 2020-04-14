package main

import (
	"encoding/json"
	"net/http"

	"github.com/Optum/dce/pkg/api"
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
	user := r.Context().Value(api.User{}).(*api.User)
	err = user.Authorize(*_lease.PrincipalID)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
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
	user := r.Context().Value(api.User{}).(*api.User)
	err = user.Authorize(*queryLease.PrincipalID)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	lease, err := Services.LeaseService().GetByAccountIDAndPrincipalID(*queryLease.AccountID, *queryLease.PrincipalID)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	deletedLease, err := Services.LeaseService().Delete(*lease.ID)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusOK, deletedLease)
}
