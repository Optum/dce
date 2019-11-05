package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Optum/dce/pkg/rolemanager"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

type createController struct {
	Dao                         db.DBer
	Queue                       common.Queue
	ResetQueueURL               string
	SNS                         common.Notificationer
	AccountCreatedTopicArn      string
	AWSSession                  session.Session
	TokenService                common.TokenService
	StoragerService             common.Storager
	RoleManager                 rolemanager.RoleManager
	PrincipalRoleName           string
	PrincipalPolicyName         string
	PrincipalMaxSessionDuration int64
	// The IAM Redbox Principal role will be denied access
	// to resources with these tags leased
	PrincipalIAMDenyTags []string
	// Tags to apply to AWS resources created by this controller
	Tags                 []*iam.Tag
	ArtifactsBucket      string
	PrincipalPolicyS3Key string
}

// Call - Function to validate the account request to add into the pool and
// publish the account creation to its respective client
func (c createController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Marshal the request JSON into a CreateRequest object
	var request createRequest
	err := json.Unmarshal([]byte(req.Body), &request)
	if err != nil {
		return response.RequestValidationError("invalid request parameters"), nil
	}

	// Validate the request body
	isValid, validationRes := request.Validate()
	if !isValid {
		return *validationRes, nil
	}

	// Check if the account already exists
	existingAccount, err := c.Dao.GetAccount(request.ID)
	if err != nil {
		log.Printf("Failed to add redbox account %s to pool: %s",
			request.ID, err.Error())
		return response.ServerError(), nil
	}
	if existingAccount != nil {
		return response.AlreadyExistsError(), nil
	}

	// Verify that we can assume role in the account,
	// using the `adminRoleArn`
	_, err = c.TokenService.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         aws.String(request.AdminRoleArn),
		RoleSessionName: aws.String("RedboxMasterAssumeRoleVerification"),
	})
	if err != nil {
		return response.RequestValidationError(
			"Unable to create Account: adminRole is not assumable by the Redbox master account",
		), nil
	}

	// Prepare the account record
	now := time.Now().Unix()
	account := db.Account{
		ID:             request.ID,
		AccountStatus:  db.NotReady,
		LastModifiedOn: now,
		CreatedOn:      now,
		AdminRoleArn:   request.AdminRoleArn,
	}

	// Create an IAM Role for the Redbox principal (end-user) to login to
	createRolRes, policyHash, err := c.createPrincipalRole(account)
	if err != nil {
		log.Printf("failed to create principal role for %s: %s", request.ID, err)
		return response.ServerError(), nil
	}
	account.PrincipalRoleArn = createRolRes.RoleArn
	account.PrincipalPolicyHash = policyHash

	// Write the Account to the DB
	err = c.Dao.PutAccount(account)
	if err != nil {
		log.Printf("Failed to add redbox account %s to pool: %s",
			request.ID, err.Error())
		return response.ServerError(), nil
	}

	// Add Account to Reset Queue
	err = c.Queue.SendMessage(&c.ResetQueueURL, &account.ID)
	if err != nil {
		log.Printf("Failed to add account %s to reset Queue: %s", account.ID, err)
		return response.ServerError(), nil
	}

	// Publish the Account to an "account-created" topic
	accountResponse := response.AccountResponse(account)
	snsMessage, err := common.PrepareSNSMessageJSON(accountResponse)
	if err != nil {
		log.Printf("Failed to create SNS account-created message for %s: %s", account.ID, err)
		return response.ServerError(), nil
	}
	_, err = c.SNS.PublishMessage(&c.AccountCreatedTopicArn, &snsMessage, true)
	if err != nil {
		log.Printf("Failed to publish SNS account-created message for %s: %s", account.ID, err)
		return response.ServerError(), nil
	}

	return response.CreateJSONResponse(
		http.StatusCreated,
		accountResponse,
	), nil
}

type createRequest struct {
	ID           string `json:"id"`
	AdminRoleArn string `json:"adminRoleArn"`
}

// Validate - Checks if the Account Request has the provided id and adminRoleArn
// fields
func (req *createRequest) Validate() (bool, *events.APIGatewayProxyResponse) {
	isValid := true
	var validationErrors []error
	if req.ID == "" {
		isValid = false
		validationErrors = append(validationErrors, errors.New("missing required field \"id\""))
	}
	if req.AdminRoleArn == "" {
		isValid = false
		validationErrors = append(validationErrors, errors.New("missing required field \"adminRoleArn\""))
	}

	if !isValid {
		errMsgs := []string{}
		for _, verr := range validationErrors {
			errMsgs = append(errMsgs, verr.Error())
		}
		msg := strings.Join(errMsgs, "; ")
		res := response.RequestValidationError(msg)
		return false, &res
	}

	return true, nil
}
