package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/db"
)

// GetLeaseByID - Returns a list of leases by principal and account
func GetLeaseByID(w http.ResponseWriter, r *http.Request) {
	// Fetch the account.
	leaseID := mux.Vars(r)["leaseID"]
	lease, err := dao.GetLeaseByID(leaseID)
	if err != nil {
		log.Printf("Error Getting Lease for Id: %s", leaseID)
		response.WriteServerErrorWithResponse(w, fmt.Sprintf("Failed Get on Lease %s", leaseID))
		return
	}
	if lease == nil {
		log.Printf("Error Getting Lease for Id: %s", err)
		response.WriteNotFoundError(w)
		return
	}

	leaseResponse := response.LeaseResponse(*lease)
	json.NewEncoder(w).Encode(leaseResponse)
}

// GetAllLeases - Returns a list of leases by principal and account
// func GetAllLeases(w http.ResponseWriter, r *http.Request) {

// }

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
	json.NewEncoder(w).Encode(leaseResponse)
}

// GetLeasesByPrincipalID - Returns a list of leases by principal and account
func GetLeasesByPrincipalID(w http.ResponseWriter, r *http.Request) {
	// Fetch the account.
	principalID := r.FormValue(AccountIDParam)
	leases, err := dao.FindLeasesByPrincipal(principalID)
	if err != nil {
		errMsg := fmt.Sprintf("Error getting leases for principal %s: %s", principalID, err.Error())
		log.Printf(errMsg)
		response.WriteServerErrorWithResponse(w, errMsg)
		return
	}

	if leases == nil || len(leases) == 0 {
		response.WriteNotFoundError(w)
		return
	}

	leaseResponses := []*response.LeaseResponse{}

	for _, l := range leases {
		leaseResponse := response.LeaseResponse(*l)
		leaseResponses = append(leaseResponses, &leaseResponse)
	}

	json.NewEncoder(w).Encode(leaseResponses)
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

	if leases == nil || len(leases) == 0 {
		// We were throwing an error on these, but not sure that's the right thing
		// to do with a REST URL with query string parameters.
		response.WriteNotFoundError(w)
		return
	}

	leaseResponses := []*response.LeaseResponse{}

	for _, l := range leases {
		leaseResponse := response.LeaseResponse(*l)
		leaseResponses = append(leaseResponses, &leaseResponse)
	}

	json.NewEncoder(w).Encode(leaseResponses)
}

// GetLeasesByStatus - Returns a list of leases by principal and account
func GetLeasesByStatus(w http.ResponseWriter, r *http.Request) {
	// Fetch the account.
	leaseStatus := r.FormValue(StatusParam)
	status, err := db.ParseLeaseStatus(leaseStatus)
	leases, err := dao.FindLeasesByStatus(status)
	if err != nil {
		errMsg := fmt.Sprintf("Error getting leases with status \"%s\": %s", leaseStatus, err.Error())
		log.Println(errMsg)
		response.WriteServerErrorWithResponse(w, errMsg)
		return
	}
	if leases == nil || len(leases) == 0 {
		response.WriteNotFoundError(w)
		return
	}

	leaseResponses := []*response.LeaseResponse{}

	for _, l := range leases {
		leaseResponse := response.LeaseResponse(*l)
		leaseResponses = append(leaseResponses, &leaseResponse)
	}

	json.NewEncoder(w).Encode(leaseResponses)
}
