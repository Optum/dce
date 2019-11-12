package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Optum/dce/pkg/rolemanager"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/gorilla/mux"
)

// DeleteAccount - Deletes the account
func DeleteAccount(w http.ResponseWriter, r *http.Request) {

	accountID := mux.Vars(r)["accountId"]
	deletedAccount, err := Dao.DeleteAccount(accountID)

	// Handle DB errors
	if err != nil {
		ErrorHandler(w, err)
		return
	}

	// Delete the IAM Principal Role for the account
	destroyIAMPrincipal(deletedAccount)

	// Publish SNS "account-deleted" message
	sendSNS(deletedAccount)

	// Push the account to the Reset Queue, so it gets cleaned up
	sendToResetQueue(deletedAccount.ID)

	// json.NewEncoder(w).Encode(response.CreateAPIResponse(http.StatusNoContent, ""))
	WriteAPIResponse(w, http.StatusNoContent, "")
}

// sendSNS sends notification to SNS that the delete has occurred.
func sendSNS(account *db.Account) {
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

func destroyIAMPrincipal(account *db.Account) {
	// Assume role into the new account
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
		log.Printf("Failed to destroy Principal IAM Role and Policy: %s", err)
	}
}
