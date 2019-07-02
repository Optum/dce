package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

type createAccountController struct {
	Dao                    db.DBer
	Queue                  common.Queue
	ResetQueueURL          string
	SNS                    common.Notificationer
	AccountCreatedTopicArn string
	AWSSession             session.Session
	TokenService           common.TokenService
}

// Call - Function to validate the account request to add into the pool and
// publish the account creation to its respective client
func (c createAccountController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Marshal the request JSON into a CreateAccountRequest object
	var request createAccountRequest
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

	// Write the Account to the DB
	now := time.Now().Unix()
	account := db.RedboxAccount{
		ID:             request.ID,
		AccountStatus:  db.NotReady,
		LastModifiedOn: now,
		CreatedOn:      now,
		AdminRoleArn:   request.AdminRoleArn,
	}
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

type createAccountRequest struct {
	ID           string `json:"id"`
	AdminRoleArn string `json:"adminRoleArn"`
}

// Validate - Checks if the Account Request has the provided id and adminRoleArn
// fields
func (req *createAccountRequest) Validate() (bool, *events.APIGatewayProxyResponse) {
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
