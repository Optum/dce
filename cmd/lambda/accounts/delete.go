package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Optum/dce/pkg/rolemanager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"

	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/gorilla/mux"
)

// DeleteAccount - Deletes the account
func DeleteAccount(w http.ResponseWriter, r *http.Request) {

	accountID := mux.Vars(r)["accountId"]

	var dao db.DBer
	if err := services.Config.GetService(&dao); err != nil {
		response.WriteServerErrorWithResponse(w, "Could not create data service")
		return
	}

	deletedAccount, err := dao.DeleteAccount(accountID)

	// Handle DB errors
	if err != nil {
		switch err.(type) {
		case *db.AccountNotFoundError:
			response.WriteNotFoundError(w)
			return
		case *db.AccountLeasedError:
			response.WriteAPIErrorResponse(
				w,
				http.StatusConflict,
				"Conflict",
				err.Error(),
			)
			return
		default:
			response.WriteServerErrorWithResponse(w, "Internal Server Error")
			return
		}
	}

	// Delete the IAM Principal Role for the account
	destroyIAMPrincipal(deletedAccount)

	// Publish SNS "account-deleted" message
	sendSNS(deletedAccount)

	// Push the account to the Reset Queue, so it gets cleaned up
	sendToResetQueue(deletedAccount.ID)

	// json.NewEncoder(w).Encode(response.CreateAPIGatewayResponse(http.StatusNoContent, ""))
	response.WriteAPIResponse(w, http.StatusNoContent, "")
}

// sendSNS sends notification to SNS that the delete has occurred.
func sendSNS(account *db.Account) error {
	serializedAccount := response.AccountResponse(*account)
	serializedMessage, err := common.PrepareSNSMessageJSON(serializedAccount)

	if err != nil {
		log.Printf("Failed to serialized SNS message for account %s: %s", account.ID, err)
		return err
	}

	var snsSvc snsiface.SNSAPI
	if err := services.Config.GetService(&snsSvc); err != nil {
		return err
	}

	_, err = snsSvc.Publish(common.CreateJSONPublishInput(&settings.AccountDeletedTopicArn, &serializedMessage))
	if err != nil {
		log.Printf("Failed to publish SNS message for account %s: %s", account.ID, err)
		return err
	}
	return nil
}

// sendToResetQueue sends the account to the reset queue
func sendToResetQueue(accountID string) error {
	var queue sqsiface.SQSAPI
	if err := services.Config.GetService(&queue); err != nil {
		return err
	}

	msgInput := common.BuildSendMessageInput(aws.String(settings.ResetQueueURL), &accountID)
	_, err := queue.SendMessage(&msgInput)
	if err != nil {
		log.Printf("Failed to add account %s to reset Queue: %s", accountID, err)
		return err
	}
	return nil
}

func destroyIAMPrincipal(account *db.Account) error {
	// Assume role into the new account
	accountSession, err := common.NewSession(services.AWSSession, account.AdminRoleArn)
	if err != nil {
		log.Printf("Failed to assume role into account %s: %s", account.ID, err)
		return err
	}
	iamClient := iam.New(accountSession)

	var roleMgr rolemanager.RoleManager

	if err := services.Config.GetService(&roleMgr); err != nil {
		log.Fatalf("Could not get role manager service")
		return err
	}

	// Destroy the role and policy
	roleMgr.SetIAMClient(iamClient)
	_, err = roleMgr.DestroyRoleWithPolicy(&rolemanager.DestroyRoleWithPolicyInput{
		RoleName:  settings.PrincipalRoleName,
		PolicyArn: fmt.Sprintf("arn:aws:iam::%s:policy/%s", account.ID, settings.PrincipalPolicyName),
	})
	// Log error, and continue
	if err != nil {
		log.Printf("Failed to destroy Principal IAM Role and Policy: %s", err)
		return err
	}
	return nil
}
