package main

import (
	"github.com/Optum/dce/pkg/api/response"
	"log"
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
	if user.Role != api.AdminGroupName && *lease.PrincipalID != user.Username {
		log.Printf("User [%s] with role: [%s] attempted to get a lease for: [%s], but was not authorized", user.Username, user.Role, *lease.PrincipalID)
		response.WriteUnauthorizedError(w)
		return
	}

	api.WriteAPIResponse(w, http.StatusOK, lease)
}
