package main

import (
	"net/http"

	"github.com/Optum/dce/pkg/api"
	"github.com/gorilla/mux"
)

// DeleteLease - Deletes the given lease
func DeleteLease(w http.ResponseWriter, r *http.Request) {

	leaseID := mux.Vars(r)["leaseID"]

	lease, err := Services.Config.LeaseService().Delete(leaseID)

	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusNoContent, lease)
}
