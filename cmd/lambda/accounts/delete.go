package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Optum/Redbox/pkg/rolemanager"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/Optum/Redbox/pkg/common"
	"github.com/Optum/Redbox/pkg/db"
	"github.com/gorilla/mux"
)

// DeleteAccount - Deletes the account
func DeleteAccount(w http.ResponseWriter, r *http.Request) {

	accountID := mux.Vars(r)["accountId"]
	deletedAccount, err := Dao.DeleteAccount(accountID)

	// Handle DB errors
	if err != nil {
		switch err.(type) {
		case *db.AccountNotFoundError:
			json.NewEncoder(w).Encode(response.NotFoundError())
		case *db.AccountLeasedError:
			json.NewEncoder(w).Encode(response.CreateAPIErrorResponse(http.StatusConflict, response.CreateErrorResponse("Conflict", err.Error())))
		default:
			json.NewEncoder(w).Encode(response.CreateAPIErrorResponse(http.StatusInternalServerError, response.CreateErrorResponse("ServerError", "Internal Server Error")))
		}
	}

	// Delete the IAM Principal Role for the account
	destroyIAMPrincipal(deletedAccount)

	// Publish SNS "account-deleted" message
	sendSNS(deletedAccount)

	// Push the account to the Reset Queue, so it gets cleaned up
	sendToResetQueue(deletedAccount.ID)

	json.NewEncoder(w).Encode(response.CreateAPIResponse(http.StatusNoContent, ""))
}

// sendSNS sends notification to SNS that the delete has occurred.
func sendSNS(account *db.RedboxAccount) {
	serializedAccount := response.AccountResponse(*account)
	serializedMessage, err := common.PrepareSNSMessageJSON(serializedAccount)

	if err != nil {
		log.Printf("Failed to serialized SNS message for account %s: %s", account.ID, err)
		return
	}

	// TODO: Probably initialize this one time at the beginning
	accountDeletedTopicArn := Config.RequireEnvVar("ACCOUNT_DELETED_TOPIC_ARN")

	_, err = SnsSvc.PublishMessage(&accountDeletedTopicArn, &serializedMessage, true)
	if err != nil {
		log.Printf("Failed to publish SNS message for account %s: %s", account.ID, err)
	}
}

// sendToResetQueue sends the account to the reset queue
func sendToResetQueue(accountID string) {
	resetQueueURL := Config.RequireEnvVar("RESET_SQS_URL")
	err := Queue.SendMessage(&resetQueueURL, &accountID)
	if err != nil {
		log.Printf("Failed to add account %s to reset Queue: %s", accountID, err)
	}
}

func destroyIAMPrincipal(account *db.RedboxAccount) {
	// Assume role into the new Redbox account
	accountSession, err := TokenSvc.NewSession(AWSSession, account.AdminRoleArn)
	if err != nil {
		log.Printf("Failed to assume role into account %s: %s", account.ID, err)
		return
	}
	iamClient := iam.New(accountSession)

	// TODO: Clean this up to initialize the following one time.
	principalRoleName := Config.RequireEnvVar("PRINCIPAL_ROLE_NAME")
	principalPolicyName := Config.RequireEnvVar("PRINCIPAL_POLICY_NAME")

	// Destroy the role and policy
	RoleManager.SetIAMClient(iamClient)
	_, err = RoleManager.DestroyRoleWithPolicy(&rolemanager.DestroyRoleWithPolicyInput{
		RoleName:  principalRoleName,
		PolicyArn: fmt.Sprintf("arn:aws:iam::%s:policy/%s", account.ID, principalPolicyName),
	})
	// Log error, and continue
	if err != nil {
		log.Printf("Failed to destroy Redbox Principal IAM Role and Policy: %s", err)
	}
}
