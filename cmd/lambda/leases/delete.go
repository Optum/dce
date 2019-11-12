package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Optum/dce/pkg/api/response"

	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
)

// requestBody is the structured object of the Request Called to the Router
type deleteLeaseRequest struct {
	PrincipalID string `json:"principalId"`
	AccountID   string `json:"accountId"`
}

var (
	queue                  common.Queue
	resetQueueURL          string
	snsService             common.Notificationer
	accountDeletedTopicArn string
)

func init() {
	accountDeletedTopicArn = Config.GetEnvVar("DECOMMISSION_TOPIC", "DefaultDecomissionTopic")
	resetQueueURL = Config.GetEnvVar("RESET_SQS_URL", "DefaultResetSQSURL")
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
	log.Printf("Decommissioning Account %s for Principal %s", accountID, principalID)

	// Move the account to decommissioned
	accts, err := Dao.FindLeasesByPrincipal(principalID)
	if err != nil {
		log.Printf("Error finding leases for Principal %s: %s", principalID, err)
		response.WriteServerErrorWithResponse(w, fmt.Sprintf("Cannot verify if Principal %s has a lease", principalID))
		return
	}
	if accts == nil {
		errStr := fmt.Sprintf("No account leases found for %s", principalID)
		log.Printf("Error: %s", errStr)
		response.WriteBadRequestError(w, errStr)
		return
	}

	// Get the Account Lease
	var acct *db.Lease
	for _, a := range accts {
		if a.AccountID == requestBody.AccountID {
			acct = a
			break
		}
	}
	if acct == nil {
		response.WriteBadRequestError(w, fmt.Sprintf("No active account leases found for %s", principalID))
		return
	} else if acct.LeaseStatus != db.Active {
		errStr := fmt.Sprintf("Account Lease is not active for %s - %s",
			principalID, accountID)
		response.WriteBadRequestError(w, errStr)
		return
	}

	// Transition the Lease Status
	updatedLease, err := Dao.TransitionLeaseStatus(acct.AccountID, principalID,
		db.Active, db.Inactive, db.LeaseDestroyed)
	if err != nil {
		log.Printf("Error transitioning lease status: %s", err)
		response.WriteServerErrorWithResponse(w, fmt.Sprintf("Failed Decommission on Account Lease %s - %s", principalID, accountID))
		return
	}

	// Transition the Account Status
	_, err = Dao.TransitionAccountStatus(acct.AccountID, db.Leased,
		db.NotReady)
	if err != nil {
		response.WriteServerErrorWithResponse(w, fmt.Sprintf("Failed Decommission on Account Lease %s - %s", principalID, accountID))
		return
	}

	leaseResponse := response.LeaseResponse(*updatedLease)
	json.NewEncoder(w).Encode(leaseResponse)
}
