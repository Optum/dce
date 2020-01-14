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
	lease, err := conf.DB.GetLeaseByID(leaseID)
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
	err = json.NewEncoder(w).Encode(leaseResponse)
	if err != nil {
		log.Printf("Error Getting Lease for Id: %s", leaseID)
		response.WriteServerErrorWithResponse(w, fmt.Sprintf("Failed Get on Lease %s", leaseID))
		return
	}
}

// GetLeasesByPrincipcalIDAndAccountID - Returns a list of leases by principal and account
func GetLeasesByPrincipcalIDAndAccountID(w http.ResponseWriter, r *http.Request) {
	// Fetch the account.
	principalID := r.FormValue(PrincipalIDParam)
	accountID := r.FormValue(AccountIDParam)
	lease, err := conf.DB.GetLease(accountID, principalID)
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
	leases, err := conf.DB.FindLeasesByPrincipal(principalID)
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
	leases, err := conf.DB.FindLeasesByAccount(accountID)
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
	leases, err := conf.DB.FindLeasesByStatus(status)
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
