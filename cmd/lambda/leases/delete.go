package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/Optum/Redbox/pkg/rolemanager"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"
)

// requestBody is the structured object of the Request Called to the Router
type deleteLeaseRequest struct {
	PrincipalID              string   `json:"principalId"`
	AccountID                string   `json:"accountId"`
	BudgetAmount             float64  `json:"budgetAmount"`
	BudgetCurrency           string   `json:"budgetCurrency"`
	BudgetNotificationEmails []string `json:"budgetNotificationEmails"`
}

type DeleteController struct {
	Dao                    db.DBer
	Queue                  common.Queue
	ResetQueueURL          string
	SNS                    common.Notificationer
	AccountDeletedTopicArn string
	AWSSession             session.Session
	TokenService           common.TokenService
	RoleManager            rolemanager.RoleManager
	PrincipalRoleName      string
	PrincipalPolicyName    string
}

func (c DeleteController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	requestBody := &deleteLeaseRequest{}

	err := json.Unmarshal([]byte(req.Body), requestBody)
	if err != nil || requestBody.PrincipalID == "" {
		log.Printf("Failed to Parse Request Body: %s", req.Body)
		return response.ClientErrorWithResponse(fmt.Sprintf("Failed to Parse Request Body: %s", req.Body)), nil
	}

	principalID := requestBody.PrincipalID
	accountID := requestBody.AccountID
	log.Printf("Decommissioning Account %s for Principal %s", accountID, principalID)

	// Move the account to decommissioned
	accts, err := c.Dao.FindLeasesByPrincipal(principalID)
	if err != nil {
		log.Printf("Error finding leases for Principal %s: %s", principalID, err)
		return response.ServerErrorWithResponse(fmt.Sprintf("Cannot verify if Principal %s has a Redbox Lease", principalID)), nil
	}
	if accts == nil {
		errStr := fmt.Sprintf("No account leases found for %s", principalID)
		log.Printf("Error: %s", errStr)
		return response.NotFoundError(), nil
	}

	// Get the Account Lease
	var acct *db.RedboxLease
	for _, a := range accts {
		if a.AccountID == requestBody.AccountID {
			acct = a
			break
		}
	}
	if acct == nil {
		return response.NotFoundError(), nil
	} else if acct.LeaseStatus != db.Active &&
		acct.LeaseStatus != db.Inactive {
		errStr := fmt.Sprintf("Account Lease is not active for %s - %s",
			principalID, accountID)
		return response.ClientErrorWithResponse(errStr), nil
	}

	// Transition the Lease Status
	_, err = c.Dao.TransitionLeaseStatus(acct.AccountID, principalID,
		db.Active, db.Inactive, "Lease deleted.")
	if err != nil {
		log.Printf("Error transitioning lease status: %s", err)
		return response.ServerErrorWithResponse(fmt.Sprintf("Failed Decommission on Account Lease %s - %s", principalID, accountID)), nil
	}

	// Transition the Account Status
	_, err = c.Dao.TransitionAccountStatus(acct.AccountID, db.Leased,
		db.NotReady)
	if err != nil {
		return response.ServerErrorWithResponse(fmt.Sprintf("Failed Decommission on Account Lease %s - %s", principalID, accountID)), nil
	}

	// return createAPIResponse(http.StatusOK, *message)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
	}, nil
}
