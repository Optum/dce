package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/aws/aws-lambda-go/events"

	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
)

// requestBody is the structured object of the Request Called to the Router
type deleteLeaseRequest struct {
	PrincipalID string `json:"principalId"`
	AccountID   string `json:"accountId"`
}

type DeleteController struct {
	Dao                    db.DBer
	Queue                  common.Queue
	ResetQueueURL          string
	SNS                    common.Notificationer
	AccountDeletedTopicArn string
}

func (c DeleteController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	requestBody := &deleteLeaseRequest{}

	err := json.Unmarshal([]byte(req.Body), requestBody)
	if err != nil || requestBody.PrincipalID == "" {
		log.Printf("Failed to Parse Request Body: %s", req.Body)
		return response.ClientBadRequestError(fmt.Sprintf("Failed to Parse Request Body: %s", req.Body)), nil
	}

	principalID := requestBody.PrincipalID
	accountID := requestBody.AccountID
	log.Printf("Destroying lease %s for Principal %s", accountID, principalID)

	// Move the account to decommissioned
	accts, err := c.Dao.FindLeasesByPrincipal(principalID)
	if err != nil {
		log.Printf("Error finding leases for Principal %s: %s", principalID, err)
		return response.ServerErrorWithResponse(fmt.Sprintf("Cannot verify if Principal %s has a lease", principalID)), nil
	}
	if accts == nil {
		errStr := fmt.Sprintf("No leases found for %s", principalID)
		log.Printf("Error: %s", errStr)
		return response.ClientBadRequestError(errStr), nil
	}

	// Get the Lease
	var acct *db.Lease
	for _, a := range accts {
		if a.AccountID == requestBody.AccountID {
			acct = a
			break
		}
	}
	if acct == nil {
		return response.ClientBadRequestError(fmt.Sprintf("No active leases found for %s", principalID)), nil
	} else if acct.LeaseStatus != db.Active {
		errStr := fmt.Sprintf("Lease is not active for %s - %s",
			principalID, accountID)
		return response.ClientBadRequestError(errStr), nil
	}

	// Transition the Lease Status
	updatedLease, err := c.Dao.TransitionLeaseStatus(acct.AccountID, principalID,
		db.Active, db.Inactive, db.LeaseDestroyed)
	if err != nil {
		log.Printf("Error transitioning lease status: %s", err)
		return response.ServerErrorWithResponse(fmt.Sprintf("Failed to destroy lease %s - %s", principalID, accountID)), nil
	}

	// Transition the Account Status
	_, err = c.Dao.TransitionAccountStatus(acct.AccountID, db.Leased,
		db.NotReady)
	if err != nil {
		return response.ServerErrorWithResponse(fmt.Sprintf("Failed to destroy lease %s - %s", principalID, accountID)), nil
	}

	leaseResponse := response.LeaseResponse(*updatedLease)
	return response.CreateApiGatewayJSONResponse(http.StatusOK, leaseResponse), nil
}
