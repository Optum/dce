package main

import (
	"net/http"

	"github.com/Optum/dce/pkg/api"
	"github.com/gorilla/mux"
)

// requestBody is the structured object of the Request Called to the Router
type deleteLeaseRequest struct {
	PrincipalID string `json:"principalId"`
	AccountID   string `json:"accountId"`
}

// DeleteLease - Deletes the given lease
func DeleteLease(w http.ResponseWriter, r *http.Request) {

	leaseID := mux.Vars(r)["leaseID"]

	lease, err := Services.Config.LeaseService().Get(leaseID)

	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}
	err = Services.Config.LeaseService().Delete(lease)

	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusNoContent, lease)

}
