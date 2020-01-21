package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/gorilla/mux"

	"github.com/Optum/dce/pkg/db"
)

// requestBody is the structured object of the Request Called to the Router
type deleteLeaseRequest struct {
	PrincipalID string `json:"principalId"`
	AccountID   string `json:"accountId"`
}

// DeleteLease - Deletes the given lease
func DeleteLease(w http.ResponseWriter, r *http.Request) {

	requestBody := &deleteLeaseRequest{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&requestBody)

	if err != nil || requestBody.PrincipalID == "" {
		log.Printf("Failed to Parse Request Body: %s", r.Body)
		response.WriteBadRequestError(w, fmt.Sprintf("Failed to Parse Request Body: %s", r.Body))
		return
	}

	principalID := requestBody.PrincipalID
	accountID := requestBody.AccountID
	log.Printf("Destroying lease %s for Principal %s", accountID, principalID)

	// Move the lease to decommissioned
	lease, err := dao.GetLease(accountID, principalID)
	if err != nil {
		log.Printf("Error finding leases for Principal %q and Account ID %q: %s", principalID, accountID, err)
		response.WriteServerErrorWithResponse(w, fmt.Sprintf("Cannot verify if Principal %q and Account %q has a lease", principalID, accountID))
		return
	}
	if lease == nil {
		errStr := fmt.Sprintf("No leases found for Principal %q and Account ID %q", principalID, accountID)
		log.Printf("Error: %s", errStr)
		response.WriteBadRequestError(w, errStr)
		return
	}

	err = endLease(w, lease)
	if err != nil {
		return
	}

	leaseResponse := response.LeaseResponse(*lease)
	err = json.NewEncoder(w).Encode(leaseResponse)
	if err != nil {
		response.WriteServerErrorWithResponse(w, fmt.Sprintf("Failed Decommission on Account Lease %s - %s", principalID, accountID))
		return
	}
}

// DeleteLeaseByID - Deletes the given lease
func DeleteLeaseByID(w http.ResponseWriter, r *http.Request) {

	leaseID := mux.Vars(r)["leaseID"]
	lease, err := dao.GetLeaseByID(leaseID)

	if err != nil {
		log.Printf("Error finding leases for ID %q: %s", leaseID, err)
		response.WriteServerErrorWithResponse(w, fmt.Sprintf("Cannot verify if Lease ID %q exists", leaseID))
		return
	}
	if lease == nil {
		errStr := fmt.Sprintf("No leases found for ID %q", leaseID)
		log.Printf("Error: %s", errStr)
		response.WriteBadRequestError(w, errStr)
		return
	}

	err = endLease(w, lease)
	if err != nil {
		return
	}

	leaseResponse := response.LeaseResponse(*lease)
	err = json.NewEncoder(w).Encode(leaseResponse)
	if err != nil {
		response.WriteServerErrorWithResponse(w, fmt.Sprintf("Failed Decommission on Account Lease %q", leaseID))
		return
	}
}

func endLease(w http.ResponseWriter, lease *db.Lease) error {
	if lease.LeaseStatus != db.Active {
		response.WriteBadRequestError(w, fmt.Sprintf("Lease is not active for %q", lease.ID))
		return fmt.Errorf("Lease is not active for %q", lease.ID)
	}

	// Transition the Lease Status
	newLease, err := dao.TransitionLeaseStatus(lease.AccountID, lease.PrincipalID,
		db.Active, db.Inactive, db.LeaseDestroyed)
	if err != nil {
		response.WriteServerErrorWithResponse(w, err.Error())
		return fmt.Errorf("Failed Decommission on Account Lease %q: %w", lease.ID, err)
	}

	*lease = *newLease

	// Transition the Account Status
	_, err = dao.TransitionAccountStatus(lease.AccountID, db.Leased,
		db.NotReady)
	if err != nil {
		response.WriteServerErrorWithResponse(w, err.Error())
		return fmt.Errorf("Failed Decommission on Account Lease %q: %w", lease.ID, err)
	}
	return nil
}
