package main

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Optum/dce/pkg/api"
)

// GetLeaseByID - Returns the single lease by ID
func GetLeaseByID(w http.ResponseWriter, r *http.Request) {

	leaseID := mux.Vars(r)["leaseID"]

	lease, err := Services.LeaseService().Get(leaseID)

	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	//If user is not an admin, they can't get leases for other users
	user := r.Context().Value(api.User{}).(*api.User)
	err = user.Authorize(*lease.PrincipalID)
	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusOK, lease)
}
