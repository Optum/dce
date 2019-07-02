package main

import (
	"context"
	"encoding/json"
	"fmt"

	"log"
	"net/http"
	"os"

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

var errorLogger = log.New(os.Stderr, "ERROR ", log.Llongfile)
var ctx = context.Background()
var appURL = "myapps.microsoft.com"

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

// publishAssignment is a helper function to create and publish an assignment
// structured message to an SNS Topic
func publishAssignment(snsSvc common.Notificationer,
	assgn *db.RedboxAccountAssignment, topic *string) (*string, error) {
	// Create a AccountAssignmentResponse based on the assgn
	assgnResp := response.CreateAccountAssignmentResponse(assgn)

	// Create the message to send to the topic from the AccountAssignment
	messageBytes, err := json.Marshal(assgnResp)
	if err != nil {
		// Rollback
		log.Printf("Error to Marshal Account Assignment: %s", err)
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
	// Assignment
	log.Printf("Sending Assignment Message to SNS Topic %s\n", *topic)
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
	UserID    string `json:"userId"`
	AccountID string `json:"accountId"`
}

// messageBody is the structured object of the JSON Message to send
// to an SNS Topic for Provision and Decommission
type messageBody struct {
	Default string `json:"default"`
	Body    string `json:"Body"`
}

// provisionAccount returns an API Gateway Response based on the execution of
// assigning a Redbox User to a Ready Redbox Account
func provisionAccount(request *requestBody, dbSvc db.DBer,
	snsSvc common.Notificationer, prov provision.Provisioner,
	topic *string) events.APIGatewayProxyResponse {
	userID := request.UserID
	log.Printf("Provisioning Account for User %s", userID)

	// Check if the users has any existing Active/FinanceLock/ResetLock
	// Assignments
	checkAssignment, err := prov.FindUserActiveAssignment(userID)
	if err != nil {
		log.Printf("Failed to Check User Active Assignments: %s", err)
		return createAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Cannot verify if User has existing Redbox Account : %s",
					err)))
	} else if checkAssignment.UserID == userID {
		errStr := fmt.Sprintf("User already has an existing Redbox: %s",
			checkAssignment.AccountID)
		log.Printf(errStr)
		return createAPIErrorResponse(http.StatusConflict,
			response.CreateErrorResponse("ClientError", errStr))
	}
	log.Printf("User %s has no Active Assignments\n", userID)

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
	log.Printf("User %s will be Assigned to Account: %s\n", userID,
		account.ID)

	// Check if the User and Account has been assigned before
	userAssignment, err := prov.FindUserAssignmentWithAccount(userID,
		account.ID)
	if err != nil {
		log.Printf("Failed to Check User Assignments with Account: %s", err)
		return createAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Cannot get Available Redbox Accounts : %s", err)))
	}

	// Create/Update a Redbox Account Assignment to Active
	create := userAssignment.AccountID == ""
	userAssignment, err = prov.ActivateAccountAssignment(create, userID,
		account.ID)
	if err != nil {
		log.Printf("Failed to Activate Account Assignment: %s", err)
		return createAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Failed to Create Assignment for Account : %s", account.ID)))
	}

	// Set the Account as assigned
	log.Printf("Set Account %s Status to Assigned for User %s\n", userID,
		account.ID)
	_, err = dbSvc.TransitionAccountStatus(account.ID, db.Ready, db.Assigned)
	if err != nil {
		// Rollback
		log.Printf("Error to Transition Account Status: %s", err)
		return rollbackProvision(prov, err, false, userID, account.ID)
	}

	// Publish Assignment to the topic
	message, err := publishAssignment(snsSvc, userAssignment, topic)
	if err != nil {
		log.Printf("Error Publish Assignment to Topic: %s", err)
		return rollbackProvision(prov, err, true, userID, account.ID)
	}

	// Return the response back to API
	return createAPIResponse(201, *message)
}

// rollbackProvision is a helper function to execute rollback for account
// provisioning
func rollbackProvision(prov provision.Provisioner, err error,
	transitionAccountStatus bool, userID string,
	accountID string) events.APIGatewayProxyResponse {
	// Attempt Rollback
	var message string
	errRollBack := prov.RollbackProvisionAccount(transitionAccountStatus,
		userID, accountID)
	if errRollBack != nil {
		log.Printf("Error to Rollback: %s", errRollBack)
		message = fmt.Sprintf("Failed to Rollback "+
			"Account Assignment for %s - %s", accountID, userID)
	} else {
		message = fmt.Sprintf("Failed to Create "+
			"Assignment for %s - %s", accountID, userID)
	}

	// Return an error
	return createAPIErrorResponse(http.StatusInternalServerError,
		response.CreateErrorResponse("ServerError", string(message)))
}

// decommissionAccount returns an API Gateway Response based on the execution of
// removing a Redbox User and setting up their Account for Reset
func decommissionAccount(request *requestBody, queueURL *string, dbSvc db.DBer,
	queue common.Queue, snsSvc common.Notificationer, topic *string) events.APIGatewayProxyResponse {
	userID := request.UserID
	accountID := request.AccountID
	log.Printf("Decommissioning Account %s for User %s", accountID, userID)

	// Move the account to decommissioned
	accts, err := dbSvc.FindAssignmentByUser(userID)
	if err != nil {
		log.Printf("Error finding assignments for User %s: %s", userID, err)
		return createAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Cannot verify if User %s has a Redbox Assignment",
					userID)))
	}
	if accts == nil {
		errStr := fmt.Sprintf("No account assignments found for %s", userID)
		log.Printf("Error: %s", errStr)
		return createAPIErrorResponse(http.StatusBadRequest,
			response.CreateErrorResponse("ClientError", errStr))
	}

	// Get the Account Assignment
	var acct *db.RedboxAccountAssignment
	for _, a := range accts {
		if a.AccountID == request.AccountID {
			acct = a
			break
		}
	}
	if acct == nil {
		errStr := fmt.Sprintf("No active account assignments found for %s", userID)
		return createAPIErrorResponse(http.StatusBadRequest,
			response.CreateErrorResponse("ClientError", errStr))
	} else if acct.AssignmentStatus != db.Active &&
		acct.AssignmentStatus != db.ResetLock {
		errStr := fmt.Sprintf("Account Assignment is not active for %s - %s",
			userID, accountID)
		return createAPIErrorResponse(http.StatusBadRequest,
			response.CreateErrorResponse("ClientError", errStr))
	}

	// Tranistion the Assignment Status
	userAssignment, err := dbSvc.TransitionAssignmentStatus(acct.AccountID, userID,
		db.Active, db.Decommissioned)
	if err != nil {
		log.Printf("Error transitioning assignment status: %s", err)
		return createAPIErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Failed Decommission on Account Assignment %s - %s", userID,
					accountID)))
	}

	// Transistion the Account Status
	_, err = dbSvc.TransitionAccountStatus(acct.AccountID, db.Assigned,
		db.NotReady)
	if err != nil {
		return createAPIErrorResponse(http.StatusInternalServerError, response.CreateErrorResponse("ServerError",
			fmt.Sprintf("Failed Decommission on Account Assignment %s - %s", userID,
				accountID)))
	}

	// Add the account to the Reset Queue
	err = queue.SendMessage(queueURL, &acct.AccountID)
	if err != nil {
		errStr := fmt.Sprintf("Failed to add Account %s to be Reset.",
			acct.AccountID)
		log.Printf("Error: %s", errStr)
		return createAPIErrorResponse(http.StatusInternalServerError, response.CreateErrorResponse("ServerError",
			fmt.Sprintf("Failed Decommission on Account Assignment %s - %s", userID,
				accountID)))
	}

	// Publish Assignment to the topic
	message, err := publishAssignment(snsSvc, userAssignment, topic)
	if err != nil {
		log.Printf("Error Publish Assignment to Topic: %s", err)
		return createAPIErrorResponse(http.StatusInternalServerError, response.CreateErrorResponse("ServerError",
			fmt.Sprintf("Failed Decommission on Account Assignment %s - %s", userID,
				accountID)))
	}

	// Return the response back to API
	return createAPIResponse(http.StatusOK, *message)
}

func router(ctx context.Context, req *events.APIGatewayProxyRequest) (
	events.APIGatewayProxyResponse, error) {
	// Extract the Body from the Request
	requestBody := &requestBody{}
	var err error
	if req.HTTPMethod != "GET" {
		err = json.Unmarshal([]byte(req.Body), requestBody)
		if err != nil || requestBody.UserID == "" {
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
	awsSession := session.New()
	snsClient := sns.New(awsSession)
	snsSvc := &common.SNS{
		Client: snsClient,
	}

	// Execute the correct action based on the HTTP method
	switch req.HTTPMethod {
	case "GET":
		// Placeholder until a proper GET gets implemented
		return createAPIResponse(http.StatusOK, "{\"message\":\"pong\"}"), nil
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

		// Set up the AWS Session
		awsSession := session.New()

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
				"Method GET/POST/DELETE are only allowed")), nil
	}
}

func main() {
	lambda.Start(router)
}
