package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/rolemanager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
)

// CreateAccount - Function to validate the account request to add into the pool and
// publish the account creation to its respective client
func CreateAccount(w http.ResponseWriter, r *http.Request) {

	// Marshal the request JSON into a CreateRequest object
	request := &CreateRequest{}
	var err error
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&request)

	if err != nil {
		response.WriteAPIErrorResponse(w, http.StatusBadRequest, "ClientError", "invalid request parameters")
		return
	}

	var tokenSvc stsiface.STSAPI
	if err := services.Config.GetService(&tokenSvc); err != nil {
		response.WriteServerErrorWithResponse(w, "Could not create token service")
		return
	}

	var dao db.DBer
	if err := services.Config.GetService(&dao); err != nil {
		response.WriteServerErrorWithResponse(w, "Could not create data service")
		return
	}

	// Set default metadata={}
	if request.Metadata == nil {
		request.Metadata = map[string]interface{}{}
	}

	// Validate the request body
	isValid, validationRes := request.Validate()
	if !isValid {
		response.WriteAPIErrorResponse(w, http.StatusBadRequest, "ClientError", *validationRes)
		return
	}

	testServices := services

	// Check if the account already exists
	existingAccount, err := dao.GetAccount(request.ID)
	if err != nil {
		log.Printf("Failed to add account %s to pool: %s",
			request.ID, err.Error())
		response.WriteAPIErrorResponse(w, http.StatusInternalServerError, "ServerError", "")
		return
	}
	if existingAccount != nil {
		response.WriteAlreadyExistsError(w)
		return
	}

	// Verify that we can assume role in the account,
	// using the `adminRoleArn`
	_, err = tokenSvc.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         aws.String(request.AdminRoleArn),
		RoleSessionName: aws.String("MasterAssumeRoleVerification"),
	})

	if err != nil {
		response.WriteRequestValidationError(
			w,
			fmt.Sprintf("Unable to add account %s to pool: adminRole is not assumable by the master account", request.ID),
		)
		return
	}

	// Prepare the account record
	now := time.Now().Unix()
	account := db.Account{
		ID:             request.ID,
		AccountStatus:  db.NotReady,
		LastModifiedOn: now,
		CreatedOn:      now,
		AdminRoleArn:   request.AdminRoleArn,
		Metadata:       request.Metadata,
	}

	// Create an IAM Role for the principal (end-user) to login to
	masterAccountID := *currentAccountID
	createRolRes, policyHash, err := createPrincipalRole(account, masterAccountID)
	if err != nil {
		log.Printf("failed to create principal role for %s: %s", request.ID, err)
		response.WriteServerErrorWithResponse(w, "Internal server error")
		return
	}
	account.PrincipalRoleArn = createRolRes.RoleArn
	account.PrincipalPolicyHash = policyHash

	// Write the Account to the DB
	err = dao.PutAccount(account)
	if err != nil {
		log.Printf("Failed to add account %s to pool: %s",
			request.ID, err.Error())
		response.WriteServerErrorWithResponse(w, "Internal server error")
		return
	}

	var queue sqsiface.SQSAPI
	if err := testServices.Config.GetService(&queue); err != nil {
		response.WriteServerErrorWithResponse(w, "Internal server error")
		return
	}
	// Add Account to Reset Queue
	msgInput := common.BuildSendMessageInput(aws.String(settings.ResetQueueURL), &account.ID)
	_, err = queue.SendMessage(&msgInput)
	if err != nil {
		log.Printf("Failed to add account %s to reset Queue: %s", account.ID, err)
		response.WriteServerErrorWithResponse(w, "Internal server error")
		return
	}

	// Publish the Account to an "account-created" topic
	accountResponse := response.AccountResponse(account)
	snsMessage, err := common.PrepareSNSMessageJSON(accountResponse)
	if err != nil {
		log.Printf("Failed to create SNS account-created message for %s: %s", account.ID, err)
		response.WriteServerErrorWithResponse(w, "Internal server error")
		return
	}

	var snsSvc snsiface.SNSAPI
	if err := services.Config.GetService(&snsSvc); err != nil {
		response.WriteServerErrorWithResponse(w, "Internal server error")
		return
	}

	_, err = snsSvc.Publish(common.CreateJSONPublishInput(&settings.AccountCreatedTopicArn, &snsMessage))
	if err != nil {
		log.Printf("Failed to publish SNS account-created message for %s: %s", account.ID, err)
		response.WriteServerErrorWithResponse(w, "Internal server error")
		return
	}

	accountResponseJSON, err := json.Marshal(accountResponse)
	if err != nil {
		log.Printf("ERROR: Failed to marshal account response for %s: %s", account.ID, err)
		response.WriteServerErrorWithResponse(w, "Internal server error")
		return
	}

	response.WriteAPIResponse(
		w,
		http.StatusCreated,
		string(accountResponseJSON),
	)
}

type CreateRequest struct {
	ID           string                 `json:"id"`
	AdminRoleArn string                 `json:"adminRoleArn"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Validate - Checks if the Account Request has the provided id and adminRoleArn
// fields
func (req *CreateRequest) Validate() (bool, *string) {
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
		return false, &msg
	}

	return true, nil
}

func createPrincipalRole(childAccount db.Account, masterAccountID string) (*rolemanager.CreateRoleWithPolicyOutput, string, error) {
	// Create an assume role policy,
	// to let principals from the same account assume the role.
	//
	// Consumers of open source DCE may modify and customize
	// this as need (eg. to integrate with SSO/SAML)
	// by responding to the "account-created" SNS topic
	assumeRolePolicy := strings.TrimSpace(fmt.Sprintf(`
		{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {
						"AWS": "arn:aws:iam::%s:root"
					},
					"Action": "sts:AssumeRole",
					"Condition": {}
				}
			]
		}
	`, masterAccountID))

	// Render the default policy for the principal
	var storageSvc common.Storager
	if err := services.Config.GetService(&storageSvc); err != nil {
		return nil, "", err
	}

	policy, policyHash, err := storageSvc.GetTemplateObject(settings.ArtifactsBucket, settings.PrincipalPolicyS3Key,
		principalPolicyInput{
			PrincipalPolicyArn:   fmt.Sprintf("arn:aws:iam::%s:policy/%s", childAccount.ID, settings.PolicyName),
			PrincipalRoleArn:     fmt.Sprintf("arn:aws:iam::%s:role/%s", childAccount.ID, settings.PrincipalRoleName),
			PrincipalIAMDenyTags: settings.PrincipalIAMDenyTags,
			AdminRoleArn:         childAccount.AdminRoleArn,
			Regions:              settings.AllowedRegions,
		})
	if err != nil {
		return nil, "", err
	}

	// Assume role into the new account
	var tokenSvc stsiface.STSAPI
	if err := services.Config.GetService(&tokenSvc); err != nil {
		return nil, "", err
	}

	accountSession, err := common.NewSession(services.AWSSession, childAccount.AdminRoleArn)
	if err != nil {
		return nil, "", err
	}
	iamClient := iam.New(accountSession)

	var roleMgr rolemanager.RoleManager
	if err := services.Config.GetService(&roleMgr); err != nil {
		return nil, "", err
	}
	// Create the Role + Policy
	roleMgr.SetIAMClient(iamClient)
	createRoleOutput := &rolemanager.CreateRoleWithPolicyOutput{}
	createRoleOutput, err = roleMgr.CreateRoleWithPolicy(&rolemanager.CreateRoleWithPolicyInput{
		RoleName:                 settings.PrincipalRoleName,
		RoleDescription:          "Role to be assumed by principal users of DCE",
		AssumeRolePolicyDocument: assumeRolePolicy,
		MaxSessionDuration:       settings.PrincipalMaxSessionDuration,
		PolicyName:               settings.PolicyName,
		PolicyDocument:           policy,
		PolicyDescription:        "Policy for principal users of DCE",
		Tags: append(settings.Tags,
			&iam.Tag{Key: aws.String("Name"), Value: aws.String("DCEPrincipal")},
		),
		IgnoreAlreadyExistsErrors: true,
	})
	return createRoleOutput, policyHash, err
}

type principalPolicyInput struct {
	PrincipalPolicyArn   string
	PrincipalRoleArn     string
	PrincipalIAMDenyTags []string
	AdminRoleArn         string
	Regions              []string
}
