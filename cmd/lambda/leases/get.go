package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/db"
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

// GetLeasesByPrincipcalIDAndAccountID - Returns a list of leases by principal and account
func GetLeasesByPrincipcalIDAndAccountID(w http.ResponseWriter, r *http.Request) {
	// Fetch the account.
	principalID := r.FormValue(PrincipalIDParam)
	accountID := r.FormValue(AccountIDParam)
	lease, err := dao.GetLease(accountID, principalID)
	if err != nil {
		errMsg := fmt.Sprintf("Error getting lease for principal %s and acccount %s: %s", principalID, accountID, err.Error())
		log.Println(errMsg)
		response.WriteServerErrorWithResponse(w, errMsg)
		return
	}
	if lease == nil {
		log.Printf("Error Getting Lease for Id: %s", err)
		response.WriteNotFoundError(w)
		return
	}

	leaseResponse := response.LeaseResponse(*lease)
	err = json.NewEncoder(w).Encode(leaseResponse)
	if err != nil {
		errMsg := fmt.Sprintf("Error getting lease for principal %s and acccount %s: %s", principalID, accountID, err.Error())
		log.Println(errMsg)
		response.WriteServerErrorWithResponse(w, errMsg)
		return
	}
}

// GetLeasesByPrincipalID - Returns a list of leases by principal and account
func GetLeasesByPrincipalID(w http.ResponseWriter, r *http.Request) {
	// Fetch the account.
	principalID := r.FormValue(PrincipalIDParam)
	leases, err := dao.FindLeasesByPrincipal(principalID)
	if err != nil {
		errMsg := fmt.Sprintf("Error getting leases for principal %s: %s", principalID, err.Error())
		log.Println(errMsg)
		response.WriteServerErrorWithResponse(w, errMsg)
		return
	}

	if len(leases) == 0 {
		response.WriteNotFoundError(w)
		return
	}

	leaseResponses := []*response.LeaseResponse{}

	for _, l := range leases {
		leaseResponse := response.LeaseResponse(*l)
		leaseResponses = append(leaseResponses, &leaseResponse)
	}

	err = json.NewEncoder(w).Encode(leaseResponses)
	if err != nil {
		errMsg := fmt.Sprintf("Error getting leases for principal %s: %s", principalID, err.Error())
		log.Println(errMsg)
		response.WriteServerErrorWithResponse(w, errMsg)
		return
	}
}

// GetLeasesByAccountID - Returns a list of leases by principal and account
func GetLeasesByAccountID(w http.ResponseWriter, r *http.Request) {
	// Fetch the account.
	accountID := r.FormValue(AccountIDParam)
	leases, err := dao.FindLeasesByAccount(accountID)
	if err != nil {
		errMsg := fmt.Sprintf("Error getting leases for account %s: %s", accountID, err.Error())
		log.Println(errMsg)
		response.WriteServerErrorWithResponse(w, errMsg)
		return
	}

	leaseResponses := []*response.LeaseResponse{}

	for _, l := range leases {
		leaseResponse := response.LeaseResponse(*l)
		leaseResponses = append(leaseResponses, &leaseResponse)
	}

	err = json.NewEncoder(w).Encode(leaseResponses)
	if err != nil {
		errMsg := fmt.Sprintf("Error getting leases for account %s: %s", accountID, err.Error())
		log.Println(errMsg)
		response.WriteServerErrorWithResponse(w, errMsg)
		return
	}
}

// GetLeasesByStatus - Returns a list of leases by lease status
func GetLeasesByStatus(w http.ResponseWriter, r *http.Request) {
	// Fetch the account.
	leaseStatus := r.FormValue(StatusParam)
	status, _ := db.ParseLeaseStatus(leaseStatus)
	leases, err := dao.FindLeasesByStatus(status)
	if err != nil {
		errMsg := fmt.Sprintf("Error getting leases with status \"%s\": %s", leaseStatus, err.Error())
		log.Println(errMsg)
		response.WriteServerErrorWithResponse(w, errMsg)
		return
	}

	if len(leases) == 0 {
		response.WriteNotFoundError(w)
		return
	}

	leaseResponses := []*response.LeaseResponse{}

	for _, l := range leases {
		leaseResponse := response.LeaseResponse(*l)
		leaseResponses = append(leaseResponses, &leaseResponse)
	}

	err = json.NewEncoder(w).Encode(leaseResponses)
	if err != nil {
		errMsg := fmt.Sprintf("Error getting leases with status \"%s\": %s", leaseStatus, err.Error())
		log.Println(errMsg)
		response.WriteServerErrorWithResponse(w, errMsg)
		return
	}
}
