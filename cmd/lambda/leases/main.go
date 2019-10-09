package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/provision"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

const (
	PrincipalIDParam     = "principalId"
	AccountIDParam       = "accountId"
	NextPrincipalIDParam = "nextPrincipalId"
	NextAccountIDParam   = "nextAccountId"
	StatusParam          = "status"
	LimitParam           = "limit"
)

// createAPIResponse is a helper function to create and return a valid response
// for an API Gateway
func createAPIResponse(status int, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: body,
	}
}

// createAPIErrorResponse is a helper function to create and return a valid error
// response message for the API
func createAPIErrorResponse(responseCode int,
	errResp response.ErrorResponse) events.APIGatewayProxyResponse {
	// Create the Error Response
	apiResponse, err := json.Marshal(errResp)

	// Should most likely not return an error since response.ErrorResponse
	// is structured to be json compatible
	if err != nil {
		log.Printf("Failed to Create Valid Error Response: %s", err)
		return createAPIResponse(http.StatusInternalServerError, fmt.Sprintf(
			"{\"error\":\"Failed to Create Valid Error Response: %s\"", err))
	}

	// Return an error
	return createAPIResponse(responseCode, string(apiResponse))
}

// publishLease is a helper function to create and publish an lease
// structured message to an SNS Topic
func publishLease(snsSvc common.Notificationer,
	assgn *db.RedboxLease, topic *string) (*string, error) {
	// Create a LeaseResponse based on the assgn
	assgnResp := response.CreateLeaseResponse(assgn)

	// Create the message to send to the topic from the Lease
	messageBytes, err := json.Marshal(assgnResp)
	if err != nil {
		// Rollback
		log.Printf("Error to Marshal Account Lease: %s", err)
		return nil, err
	}
	message := string(messageBytes)

	// Create the messageBody to make it compatible with SNS JSON
	provBody := messageBody{
		Default: message,
		Body:    message,
	}
	provMessageBytes, err := json.Marshal(provBody)
	if err != nil {
		// Rollback
		log.Printf("Error to Marshal Message Body: %s", err)
		return nil, err
	}
	provMessage := string(provMessageBytes)

	// Publish message to the Provision Topic on the success of the Account
	// Lease
	log.Printf("Sending Lease Message to SNS Topic %s\n", *topic)
	messageID, err := snsSvc.PublishMessage(topic, &provMessage, true)
	if err != nil {
		// Rollback
		log.Printf("Error to Send Message to SNS Topic %s: %s", *topic, err)
		return nil, err
	}
	log.Printf("Success Message Sent to SNS Topic %s: %s\n", *topic, *messageID)
	return &message, nil
}

// requestBody is the structured object of the Request Called to the Router
type requestBody struct {
	PrincipalID              string   `json:"principalId"`
	AccountID                string   `json:"accountId"`
	BudgetAmount             float64  `json:"budgetAmount"`
	BudgetCurrency           string   `json:"budgetCurrency"`
	BudgetNotificationEmails []string `json:"budgetNotificationEmails"`
	RequestedLeastEnd        int64    `json:"requestedLeaseEnd"`
}

// messageBody is the structured object of the JSON Message to send
// to an SNS Topic for Provision and Decommission
type messageBody struct {
	Default string `json:"default"`
	Body    string `json:"Body"`
}

// provisionAccount returns an API Gateway Response based on the execution of
// leasing a Redbox Principal to a Ready Redbox Account
func provisionAccount(request *requestBody, dbSvc db.DBer,
	snsSvc common.Notificationer, prov provision.Provisioner,
	topic *string) events.APIGatewayProxyResponse {
	principalID := request.PrincipalID
	log.Printf("Provisioning Account for Principal %s", principalID)

	// Just do a quick sanity check on the request and make sure that the
	// requested lease end date, if specified, is at least greater than
	// today and if it isn't then return an error response
	if request.RequestedLeastEnd != 0 && request.RequestedLeastEnd <= time.Now().Unix() {
		errStr := fmt.Sprintf("Requested lease has a desired expiry date less than today: %d", request.RequestedLeastEnd)
		log.Printf(errStr)
		return createAPIErrorResponse(http.StatusBadRequest,
			response.CreateErrorResponse("ClientError", errStr))
	}

	// Check if the principal has any existing Active/FinanceLock/ResetLock
	// Leases
	checkLease, err := prov.FindActiveLeaseForPrincipal(principalID)
	if err != nil {
		log.Printf("Failed to Check Principal Active Leases: %s", err)
		return createAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Cannot verify if Principal has existing Redbox Account : %s",
					err)))
	} else if checkLease.PrincipalID == principalID {
		errStr := fmt.Sprintf("Principal already has an existing Redbox: %s",
			checkLease.AccountID)
		log.Printf(errStr)
		return createAPIErrorResponse(http.StatusConflict,
			response.CreateErrorResponse("ClientError", errStr))
	}
	log.Printf("Principal %s has no Active Leases\n", principalID)

	// Get the First Ready Account
	// Exit if there's an error or no ready accounts
	account, err := dbSvc.GetReadyAccount()
	if err != nil {
		log.Printf("Failed to Check Ready Accounts: %s", err)
		return createAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Cannot get Available Redbox Accounts : %s", err)))
	} else if account == nil {
		errStr := "No Available Redbox Accounts at this moment"
		log.Printf(errStr)
		return createAPIErrorResponse(http.StatusServiceUnavailable,
			response.CreateErrorResponse("ServerError", errStr))
	}
	log.Printf("Principal %s will be Leased to Account: %s\n", principalID,
		account.ID)

	// Check if the Principal and Account has been leased before
	lease, err := prov.FindLeaseWithAccount(principalID,
		account.ID)
	if err != nil {
		log.Printf("Failed to Check Leases with Account: %s", err)
		return createAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Cannot get Available Redbox Accounts : %s", err)))
	}

	// Create/Update a Redbox Account Lease to Active
	create := lease.AccountID == ""
	lease, err = prov.ActivateAccount(create, principalID,
		account.ID, request.BudgetAmount, request.BudgetCurrency, request.BudgetNotificationEmails, request.RequestedLeastEnd)
	if err != nil {
		log.Printf("Failed to Activate Account Lease: %s", err)
		return createAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Failed to Create Lease for Account : %s", account.ID)))
	}

	// Set the Account as leased
	log.Printf("Set Account %s Status to Leased for Principal %s\n", principalID,
		account.ID)
	_, err = dbSvc.TransitionAccountStatus(account.ID, db.Ready, db.Leased)
	if err != nil {
		// Rollback
		log.Printf("Error to Transition Account Status: %s", err)
		return rollbackProvision(prov, err, false, principalID, account.ID)
	}

	// Publish Lease to the topic
	message, err := publishLease(snsSvc, lease, topic)
	if err != nil {
		log.Printf("Error Publish Lease to Topic: %s", err)
		return rollbackProvision(prov, err, true, principalID, account.ID)
	}

	// Return the response back to API
	return createAPIResponse(201, *message)
}

// rollbackProvision is a helper function to execute rollback for account
// provisioning
func rollbackProvision(prov provision.Provisioner, err error,
	transitionAccountStatus bool, principalID string,
	accountID string) events.APIGatewayProxyResponse {
	// Attempt Rollback
	var message string
	errRollBack := prov.RollbackProvisionAccount(transitionAccountStatus,
		principalID, accountID)
	if errRollBack != nil {
		log.Printf("Error to Rollback: %s", errRollBack)
		message = fmt.Sprintf("Failed to Rollback "+
			"Account Lease for %s - %s", accountID, principalID)
	} else {
		message = fmt.Sprintf("Failed to Create "+
			"Lease for %s - %s", accountID, principalID)
	}

	// Return an error
	return createAPIErrorResponse(http.StatusInternalServerError,
		response.CreateErrorResponse("ServerError", string(message)))
}

// decommissionAccount returns an API Gateway Response based on the execution of
// removing a Redbox Principal and setting up their Account for Reset
func decommissionAccount(request *requestBody, queueURL *string, dbSvc db.DBer,
	queue common.Queue, snsSvc common.Notificationer, topic *string) events.APIGatewayProxyResponse {
	principalID := request.PrincipalID
	accountID := request.AccountID
	log.Printf("Decommissioning Account %s for Principal %s", accountID, principalID)

	// Move the account to decommissioned
	accts, err := dbSvc.FindLeasesByPrincipal(principalID)
	if err != nil {
		log.Printf("Error finding leases for Principal %s: %s", principalID, err)
		return createAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Cannot verify if Principal %s has a Redbox Lease",
					principalID)))
	}
	if accts == nil {
		errStr := fmt.Sprintf("No account leases found for %s", principalID)
		log.Printf("Error: %s", errStr)
		return createAPIErrorResponse(http.StatusBadRequest,
			response.CreateErrorResponse("ClientError", errStr))
	}

	// Get the Account Lease
	var acct *db.RedboxLease
	for _, a := range accts {
		if a.AccountID == request.AccountID {
			acct = a
			break
		}
	}
	if acct == nil {
		errStr := fmt.Sprintf("No active account leases found for %s", principalID)
		return createAPIErrorResponse(http.StatusBadRequest,
			response.CreateErrorResponse("ClientError", errStr))
	} else if acct.LeaseStatus != db.Active {
		errStr := fmt.Sprintf("Account Lease is not active for %s - %s",
			principalID, accountID)
		return createAPIErrorResponse(http.StatusBadRequest,
			response.CreateErrorResponse("ClientError", errStr))
	}

	// Transition the Lease Status
	lease, err := dbSvc.TransitionLeaseStatus(acct.AccountID, principalID,
		db.Active, db.Inactive, "Requested decommission.")
	if err != nil {
		log.Printf("Error transitioning lease status: %s", err)
		return createAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Failed Decommission on Account Lease %s - %s", principalID,
					accountID)))
	}

	// Transition the Account Status
	_, err = dbSvc.TransitionAccountStatus(acct.AccountID, db.Leased,
		db.NotReady)
	if err != nil {
		return createAPIErrorResponse(http.StatusInternalServerError, response.CreateErrorResponse("ServerError",
			fmt.Sprintf("Failed Decommission on Account Lease %s - %s", principalID,
				accountID)))
	}

	// Add the account to the Reset Queue
	err = queue.SendMessage(queueURL, &acct.AccountID)
	if err != nil {
		errStr := fmt.Sprintf("Failed to add Account %s to be Reset.",
			acct.AccountID)
		log.Printf("Error: %s", errStr)
		return createAPIErrorResponse(http.StatusInternalServerError, response.CreateErrorResponse("ServerError",
			fmt.Sprintf("Failed Decommission on Account Lease %s - %s", principalID,
				accountID)))
	}

	// Publish Lease to the topic
	message, err := publishLease(snsSvc, lease, topic)
	if err != nil {
		log.Printf("Error Publish Lease to Topic: %s", err)
		return createAPIErrorResponse(http.StatusInternalServerError, response.CreateErrorResponse("ServerError",
			fmt.Sprintf("Failed Decommission on Account Lease %s - %s", principalID,
				accountID)))
	}

	// Return the response back to API
	return createAPIResponse(http.StatusOK, *message)
}

// parseGetLeasesInput creates a GetLeasesInput from the query parameters
func parseGetLeasesInput(queryParams map[string]string) (db.GetLeasesInput, error) {
	query := db.GetLeasesInput{
		StartKeys: make(map[string]string),
	}

	status, ok := queryParams[StatusParam]
	if ok && len(status) > 0 {
		query.Status = status
	}

	limit, ok := queryParams[LimitParam]
	if ok && len(limit) > 0 {
		limInt, err := strconv.ParseInt(limit, 10, 64)
		query.Limit = limInt
		if err != nil {
			return query, err
		}
	}

	principalID, ok := queryParams[PrincipalIDParam]
	if ok && len(principalID) > 0 {
		query.PrincipalID = principalID
	}

	accountID, ok := queryParams[AccountIDParam]
	if ok && len(accountID) > 0 {
		query.AccountID = accountID
	}

	nextAccountID, ok := queryParams[NextAccountIDParam]
	if ok && len(nextAccountID) > 0 {
		query.StartKeys["AccountId"] = nextAccountID
	}

	nextPrincipalID, ok := queryParams[NextPrincipalIDParam]
	if ok && len(nextPrincipalID) > 0 {
		query.StartKeys["PrincipalId"] = nextPrincipalID
	}

	return query, nil
}

// buildBaseURL returns a base API url from the request properties.
func buildBaseURL(req *events.APIGatewayProxyRequest) string {
	return fmt.Sprintf("https://%s/%s", req.Headers["Host"], req.RequestContext.Stage)
}

// buildNextURL merges the next parameters into the request parameters and returns an API URL.
func buildNextURL(req *events.APIGatewayProxyRequest, nextParams map[string]string) string {
	responseParams := make(map[string]string)
	responseQueryStrings := make([]string, 0)
	base := buildBaseURL(req)

	for k, v := range req.QueryStringParameters {
		responseParams[k] = v
	}

	for k, v := range nextParams {
		responseParams[fmt.Sprintf("next%s", k)] = v
	}

	for k, v := range responseParams {
		responseQueryStrings = append(responseQueryStrings, fmt.Sprintf("%s=%s", k, v))
	}

	queryString := strings.Join(responseQueryStrings, "&")
	return fmt.Sprintf("%s%s?%s", base, req.Path, queryString)
}

func router(ctx context.Context, req *events.APIGatewayProxyRequest) (
	events.APIGatewayProxyResponse, error) {
	// Extract the Body from the Request
	requestBody := &requestBody{}
	var err error
	if req.HTTPMethod != "GET" {
		err = json.Unmarshal([]byte(req.Body), requestBody)
		if err != nil || requestBody.PrincipalID == "" {
			log.Printf("Failed to Parse Request Body: %s", req.Body)
			return createAPIErrorResponse(http.StatusBadRequest,
				response.CreateErrorResponse("ClientError",
					fmt.Sprintf("Failed to Parse Request Body: %s", req.Body))), nil
		}
	}

	// Create the Database Service from the environment
	dbSvc, err := db.NewFromEnv()
	if err != nil {
		log.Printf("Failed to Initialize Database: %s", err)
		return createAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse(
				"ServerError", "Failed Database Initialization")), nil
	}

	// Create the SNS Service
	awsSession, err := session.NewSession()
	if err != nil {
		return response.ServerErrorWithResponse(fmt.Sprintf("Error creating AWS session: %s", err)), nil
	}

	snsClient := sns.New(awsSession)
	snsSvc := &common.SNS{
		Client: snsClient,
	}

	// Execute the correct action based on the HTTP method
	switch req.HTTPMethod {
	case "GET":
		getLeasesInput, err := parseGetLeasesInput(req.QueryStringParameters)

		if err != nil {
			return response.RequestValidationError(fmt.Sprintf("Error parsing query params")), nil
		}

		result, err := dbSvc.GetLeases(getLeasesInput)

		if err != nil {
			return response.ServerErrorWithResponse(fmt.Sprintf("Error querying leases: %s", err)), nil
		}

		// Convert DB Lease model to API Response model
		leaseResponseItems := []response.LeaseResponse{}
		for _, lease := range result.Results {
			leaseResponseItems = append(leaseResponseItems, response.LeaseResponse(*lease))
		}

		responseBytes, err := json.Marshal(leaseResponseItems)

		if err != nil {
			return response.ServerErrorWithResponse(fmt.Sprintf("Error serializing response: %s", err)), nil
		}

		res := createAPIResponse(http.StatusOK, string(responseBytes))

		// If the DB result has next keys, then the URL to retrieve the next page is put into the Link header.
		if len(result.NextKeys) > 0 {
			nextURL := buildNextURL(req, result.NextKeys)
			res.Headers["Link"] = fmt.Sprintf("<%s>; rel=\"next\"", nextURL)
		}

		return res, nil

	case "POST":
		prov := &provision.AccountProvision{
			DBSvc: dbSvc,
		}
		topic := common.RequireEnv("PROVISION_TOPIC")

		return provisionAccount(requestBody, dbSvc, snsSvc, prov, &topic), nil
	case "DELETE":
		topic := common.RequireEnv("DECOMMISSION_TOPIC")
		// Verify the request body provides the AccountID
		if requestBody.AccountID == "" {
			log.Printf("Failed to Parse Account ID from Request Body: %s",
				req.Body)
			return createAPIErrorResponse(http.StatusBadRequest,
				response.CreateErrorResponse("ClientError",
					fmt.Sprintf("Failed to Parse Accountr ID Request Body: %s",
						req.Body))), nil
		}

		// Get the reset queue url
		queueURL := common.RequireEnv("RESET_SQS_URL")

		// Construct a Queue
		sqsClient := sqs.New(awsSession)
		queue := common.SQSQueue{
			Client: sqsClient,
		}

		return decommissionAccount(requestBody, &queueURL, dbSvc, queue,
			snsSvc, &topic), nil
	default:
		return createAPIErrorResponse(http.StatusMethodNotAllowed,
			response.CreateErrorResponse("ClientError",
				"Methods GET/POST/DELETE are only allowed")), nil
	}
}

func main() {
	lambda.Start(router)
}
