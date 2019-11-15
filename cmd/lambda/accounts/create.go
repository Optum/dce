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
	"github.com/aws/aws-sdk-go/service/sts"
)

var (
	accountCreatedTopicArn      string
	policyName                  string
	artifactsBucket             string
	principalPolicyS3Key        string
	principalRoleName           string
	principalIAMDenyTags        []string
	principalMaxSessionDuration int64
	tags                        []*iam.Tag
	resetQueueURL               string
)

func init() {
	policyName = Config.GetEnvVar("PRINCIPAL_POLICY_NAME", "DCEPrincipalDefaultPolicy")
	artifactsBucket = Config.GetEnvVar("ARTIFACTS_BUCKET", "DefaultArtifactBucket")
	principalPolicyS3Key = Config.GetEnvVar("PRINCIPAL_POLICY_S3_KEY", "DefaultPrincipalPolicyS3Key")
	principalRoleName = Config.GetEnvVar("PRINCIPAL_ROLE_NAME", "DCEPrincipal")
	principalIAMDenyTags = strings.Split(Config.GetEnvVar("PRINCIPAL_IAM_DENY_TAGS", "DefaultPrincipalIamDenyTags"), ",")
	principalMaxSessionDuration = int64(Config.GetEnvIntVar("PRINCIPAL_MAX_SESSION_DURATION", 100))
	tags = []*iam.Tag{
		{Key: aws.String("Terraform"), Value: aws.String("False")},
		{Key: aws.String("Source"), Value: aws.String("github.com/Optum/dce//cmd/lambda/accounts")},
		{Key: aws.String("Environment"), Value: aws.String(Config.GetEnvVar("TAG_ENVIRONMENT", "DefaultTagEnvironment"))},
		{Key: aws.String("Contact"), Value: aws.String(Config.GetEnvVar("TAG_CONTACT", "DefaultTagContact"))},
		{Key: aws.String("AppName"), Value: aws.String(Config.GetEnvVar("TAG_APP_NAME", "DefaultTagAppName"))},
	}
	accountCreatedTopicArn = Config.GetEnvVar("ACCOUNT_CREATED_TOPIC_ARN", "DefaultAccountCreatedTopicArn")
	resetQueueURL = Config.GetEnvVar("RESET_SQS_URL", "DefaultResetSQSUrl")
}

// CreateAccount - Function to validate the account request to add into the pool and
// publish the account creation to its respective client
func CreateAccount(w http.ResponseWriter, r *http.Request) {

	// Marshal the request JSON into a CreateRequest object
	request := &CreateRequest{}
	var err error
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&request)

	if err != nil {
		WriteAPIErrorResponse(w, http.StatusBadRequest, "ClientError", "invalid request parameters")
		return
	}

	// Validate the request body
	isValid, validationRes := request.Validate()
	if !isValid {
		WriteAPIErrorResponse(w, http.StatusBadRequest, "ClientError", *validationRes)
		return
	}

	// Check if the account already exists
	existingAccount, err := Dao.GetAccount(request.ID)
	if err != nil {
		log.Printf("Failed to add account %s to pool: %s",
			request.ID, err.Error())
		WriteAPIErrorResponse(w, http.StatusInternalServerError, "ServerError", "")
		return
	}
	if existingAccount != nil {
		WriteAlreadyExistsError(w)
		return
	}

	// Verify that we can assume role in the account,
	// using the `adminRoleArn`
	_, err = TokenSvc.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         aws.String(request.AdminRoleArn),
		RoleSessionName: aws.String("MasterAssumeRoleVerification"),
	})

	if err != nil {
		WriteRequestValidationError(
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
	}

	// Create an IAM Role for the principal (end-user) to login to
	masterAccountID := *CurrentAccountID
	createRolRes, policyHash, err := createPrincipalRole(account, masterAccountID)
	if err != nil {
		log.Printf("failed to create principal role for %s: %s", request.ID, err)
		WriteServerErrorWithResponse(w, "Internal server error")
		return
	}
	account.PrincipalRoleArn = createRolRes.RoleArn
	account.PrincipalPolicyHash = policyHash

	// Write the Account to the DB
	err = Dao.PutAccount(account)
	if err != nil {
		log.Printf("Failed to add account %s to pool: %s",
			request.ID, err.Error())
		WriteServerErrorWithResponse(w, "Internal server error")
		return
	}

	// Add Account to Reset Queue
	err = Queue.SendMessage(&resetQueueURL, &account.ID)
	if err != nil {
		log.Printf("Failed to add account %s to reset Queue: %s", account.ID, err)
		WriteServerErrorWithResponse(w, "Internal server error")
		return
	}

	// Publish the Account to an "account-created" topic
	accountResponse := response.AccountResponse(account)
	snsMessage, err := common.PrepareSNSMessageJSON(accountResponse)
	if err != nil {
		log.Printf("Failed to create SNS account-created message for %s: %s", account.ID, err)
		WriteServerErrorWithResponse(w, "Internal server error")
		return
	}

	// TODO: Initialize these in a better spot.

	_, err = SnsSvc.PublishMessage(&accountCreatedTopicArn, &snsMessage, true)
	if err != nil {
		log.Printf("Failed to publish SNS account-created message for %s: %s", account.ID, err)
		WriteServerErrorWithResponse(w, "Internal server error")
		return
	}

	accountResponseJSON, err := json.Marshal(accountResponse)

	WriteAPIResponse(
		w,
		http.StatusCreated,
		string(accountResponseJSON),
	)
}

// CreateRequest - The create account request
type CreateRequest struct {
	ID           string `json:"id"`
	AdminRoleArn string `json:"adminRoleArn"`
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

	policy, policyHash, err := StorageSvc.GetTemplateObject(artifactsBucket, principalPolicyS3Key,
		principalPolicyInput{
			PrincipalPolicyArn:   fmt.Sprintf("arn:aws:iam::%s:policy/%s", childAccount.ID, policyName),
			PrincipalRoleArn:     fmt.Sprintf("arn:aws:iam::%s:role/%s", childAccount.ID, principalRoleName),
			PrincipalIAMDenyTags: principalIAMDenyTags,
			AdminRoleArn:         childAccount.AdminRoleArn,
		})
	if err != nil {
		return nil, "", err
	}

	// Assume role into the new account
	accountSession, err := TokenSvc.NewSession(AWSSession, childAccount.AdminRoleArn)
	if err != nil {
		return nil, "", err
	}
	iamClient := iam.New(accountSession)

	// Create the Role + Policy
	RoleManager.SetIAMClient(iamClient)
	createRoleOutput := &rolemanager.CreateRoleWithPolicyOutput{}
	createRoleOutput, err = RoleManager.CreateRoleWithPolicy(&rolemanager.CreateRoleWithPolicyInput{
		RoleName:                 principalRoleName,
		RoleDescription:          "Role to be assumed by principal users of DCE",
		AssumeRolePolicyDocument: assumeRolePolicy,
		MaxSessionDuration:       principalMaxSessionDuration,
		PolicyName:               policyName,
		PolicyDocument:           policy,
		PolicyDescription:        "Policy for principal users of DCE",
		Tags: append(tags,
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
}
