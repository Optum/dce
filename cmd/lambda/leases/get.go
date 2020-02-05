package main

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Optum/dce/pkg/api"
)

// GetLeaseByID - Returns the single lease by ID
func GetLeaseByID(w http.ResponseWriter, r *http.Request) {

	leaseID := mux.Vars(r)["leaseID"]

	lease, err := Services.Config.LeaseService().Get(leaseID)

	if err != nil {
		api.WriteAPIErrorResponse(w, err)
		return
	}

	api.WriteAPIResponse(w, http.StatusOK, lease)
}
