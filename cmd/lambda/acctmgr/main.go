package main

import (
	"context"
	"errors"
	"fmt"

	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	azureauth "github.com/Optum/Redbox/pkg/authorization"
	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/provision"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var errorLogger = log.New(os.Stderr, "ERROR ", log.Llongfile)
var ctx = context.Background()
var appURL = "myapps.microsoft.com"

func checkGroupMembership(ck *common.ClaimKey, dbSvc db.DBer,
	auth azureauth.Authorizationer) (events.APIGatewayProxyResponse, error) {
	// Find users assigned AWS account
	assignedAccts, err := dbSvc.FindAssignmentByUser(ck.UserID)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       string("Failed Query User Assignment"),
		}, err
	}

	// Associate AWS account to GroupID in Azure AD
	acctInfo, err := dbSvc.GetAccount(assignedAccts[0].AccountID)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       string("Failed Query Account Info"),
		}, err
	}

	// Query Graph AD for user presence in group
	chkGrpMemResult, err := auth.ADGroupMember(ctx, &acctInfo.GroupID,
		&ck.UserID, &ck.TenantID)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 403,
			Body:       string("Group check failed"),
		}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(strconv.FormatBool(chkGrpMemResult)),
	}, nil
}

// provisionAccount returns an API Gateway Response based on the execution of
// assigning a Redbox User to a Ready Redbox Account
func provisionAccount(ck common.ClaimKey, dbSvc db.DBer, prov provision.Provisioner,
	auth azureauth.Authorizationer) (events.APIGatewayProxyResponse, error) {
	userID := ck.UserID
	tenantID := ck.TenantID
	log.Printf("Provisioning Account for User %s", userID)

	// Check if the users has any existing Active/FinanceLock/ResetLock
	// Assignments
	checkAssignment, err := prov.FindUserActiveAssignment(userID)
	if err != nil {
		return createResponse(503, fmt.Sprintf("Cannot verify if User has "+
			"existing Redbox Account : %s", err)), err
	} else if checkAssignment.UserID == userID {
		errStr := fmt.Sprintf("User already has an existing Redbox: %s",
			checkAssignment.AccountID)
		return createResponse(409, errStr), errors.New(errStr)
	}
	log.Printf("User %s has no Active Assignments\n", userID)

	// Get the First Ready Account
	// Exit if there's an error or no ready accounts
	account, err := dbSvc.GetReadyAccount()
	if err != nil {
		return createResponse(503, fmt.Sprintf("Cannot get Available Redbox "+
			"Accounts : %s", err)), err
	} else if account == nil {
		errStr := "No Available Redbox Accounts at this moment"
		return createResponse(503, errStr), errors.New(errStr)
	}
	log.Printf("User %s will be Assigned to Account: %s\n", userID, account.ID)

	// Check if the User and Account has been assigned before
	userAssignment, err := prov.FindUserAssignmentWithAccount(userID,
		account.ID)
	if err != nil {
		return createResponse(503, fmt.Sprintf("Cannot get Available Redbox "+
			"Accounts : %s", err)), err
	}

	// Create/Update a Redbox Account Assignment to Active
	create := userAssignment.AccountID == ""
	err = prov.ActivateAccountAssignment(create, userID, account.ID)
	if err != nil {
		return createResponse(500, fmt.Sprintf("Failed to Create "+
			"Assignment for Account : %s", account.ID)), err
	}

	// Set the Account as assigned
	log.Printf("Set Account %s Status to Ready for User %s\n", userID,
		account.ID)
	_, err = dbSvc.TransitionAccountStatus(account.ID, db.Ready, db.Assigned)
	if err != nil {
		// Rollback
		errRollBack := prov.RollbackProvisionAccount(false, userID, account.ID)
		if errRollBack != nil {
			return createResponse(500, fmt.Sprintf("Failed to Rollback "+
				"Account Assignment for Account : %s", account.ID)), errRollBack
		}
		// Return an error
		return createResponse(500, fmt.Sprintf("Failed to Create "+
			"Assignment for Account : %s", account.ID)), err
	}

	// Add the AD Group User to the AD Group Account
	log.Printf("Add User %s to AD Group %s for Account %s\n", userID,
		account.GroupID, account.ID)
	group, err := auth.AddADGroupUser(ctx, userID, account.GroupID, tenantID)
	if err != nil || group.StatusCode != 204 {
		// Rollback
		errRollBack := prov.RollbackProvisionAccount(true, userID, account.ID)
		if errRollBack != nil {
			return createResponse(500, fmt.Sprintf("Failed to Rollback "+
				"Account Assignment for Account : %s", account.ID)), errRollBack
		}
		// Return an error
		return createResponse(500, fmt.Sprintf("Fail to Add User %s for Account : %s",
			userID, account.ID)), err
	}

	// Return the response back to API
	return createResponse(201, fmt.Sprintf("User successfully added to group "+
		"and Redbox account manifest has been updated. Your AWS account is "+
		"%s. To login, please go to %s.", account.ID, appURL)), nil
}

// createResponse is a helper function to create and return a valid response
// for an API Gateway
func createResponse(status int, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       body,
	}
}

// decommissionAccount returns an API Gateway Response based on the execution of
// removing a Redbox User and setting up their Account for Reset
func decommissionAccount(ck *common.ClaimKey, queueURL *string, dbSvc db.DBer,
	queue common.Queue, auth azureauth.Authorizationer) (
	events.APIGatewayProxyResponse, error) {
	// Move the account to decommissioned
	accts, err := dbSvc.FindAssignmentByUser(ck.UserID)
	if err != nil {
		return createResponse(503, fmt.Sprintf("Cannot verify if User has "+
			"existing Redbox Account : %s", err)), err
	}
	if accts == nil {
		errStr := fmt.Sprintf("No account assignments found for %s", ck.UserID)
		return createResponse(400, errStr), errors.New(errStr)
	}

	var acct *db.RedboxAccountAssignment
	for _, a := range accts {
		if a.AssignmentStatus == db.Active {
			acct = a
			break
		}
	}

	if acct == nil {
		errStr := fmt.Sprintf("No active account assignments found for %s", ck.UserID)
		return createResponse(400, errStr), errors.New(errStr)
	}

	_, err = dbSvc.TransitionAssignmentStatus(acct.AccountID, ck.UserID,
		db.Active, db.Decommissioned)
	if err != nil {
		return createResponse(500, "Failed Decommission on Account Assignment"),
			err
	}
	var acctFin *db.RedboxAccount
	acctFin, err = dbSvc.TransitionAccountStatus(acct.AccountID, db.Assigned,
		db.NotReady)
	if err != nil {
		return createResponse(500, "Failed Decommission on Account"), err
	}

	// After Dynamo succeeds, actually remove user from group in Graph AD in Azure
	response, err := auth.RemoveADGroupUser(ctx, acctFin.GroupID, ck.UserID,
		ck.TenantID)
	if err != nil || response.StatusCode != 204 {
		return createResponse(500, fmt.Sprintf("User has not been removed "+
			"from group '%s'.", acctFin.GroupID)), err
	}

	// Add the account to the Reset Queue
	err = queue.SendMessage(queueURL, &acct.AccountID)
	if err != nil {
		return createResponse(500, fmt.Sprintf("Failed to add Account %s to "+
			"be Reset.", acct.AccountID)), err
	}

	return createResponse(http.StatusOK, fmt.Sprintf("AWS Redbox "+
		"Decommission: User '%s' has been removed from the account group '%s'.",
		ck.UserID, acctFin.GroupID)), nil
}

func router(ctx context.Context, req *events.APIGatewayProxyRequest) (
	events.APIGatewayProxyResponse, error) {
	// Verify the Request and initialize the Claim Key
	jwtToken := &common.JWT{
		Claims: &common.ClaimKey{},
	}
	jwtToken.Req = req
	err := jwtToken.ParseJWT()
	if err != nil {
		return createResponse(403, "Failed Parse on JWT"), err
	}
	if jwtToken.Claims.UserID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: 403,
			Body:       string("Failed JWT Verification"),
		}, errors.New("Failed JWT verification")
	}

	// Create the Database Service from the environment
	dbSvc, err := db.NewFromEnv()
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       string("Failed Database Initialization"),
		}, err
	}

	// Create the AzureAuthorization to be used in the respective calls
	auth := azureauth.AzureAuthorization{}

	// Execute the correct action based on the HTTP method
	switch req.HTTPMethod {
	case "GET":
		return checkGroupMembership(jwtToken.Claims, dbSvc, &auth)
	case "POST":
		prov := provision.AccountProvision{
			DBSvc: dbSvc,
		}
		return provisionAccount(*jwtToken.Claims, dbSvc, &prov, &auth)
	case "DELETE":
		// Get the reset queue url
		queueURL := common.RequireEnv("RESET_SQS_URL")

		// Set up the AWS Session
		awsSession := session.New()

		// Construct a Queue
		sqsClient := sqs.New(awsSession)
		queue := common.SQSQueue{
			Client: sqsClient,
		}

		return decommissionAccount(jwtToken.Claims, &queueURL, dbSvc, queue, &auth)
	default:
		return createResponse(http.StatusMethodNotAllowed,
			"Method get/post/put are only allowed"), nil
	}
}

func main() {
	lambda.Start(router)
}
