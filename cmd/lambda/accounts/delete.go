package main

import (
	"context"
	"fmt"
	"github.com/Optum/Dce/pkg/rolemanager"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"log"
	"net/http"
	"path"

	"github.com/Optum/Dce/pkg/api/response"
	"github.com/Optum/Dce/pkg/common"
	"github.com/Optum/Dce/pkg/db"
	"github.com/aws/aws-lambda-go/events"
)

type deleteController struct {
	Dao                    db.DBer
	Queue                  common.Queue
	ResetQueueURL          string
	SNS                    common.Notificationer
	AccountDeletedTopicArn string
	AWSSession             session.Session
	TokenService           common.TokenService
	RoleManager            rolemanager.RoleManager
	PrincipalRoleName      string
	PrincipalPolicyName    string
}

// Call handles DELETE /accounts/{id} requests. Returns no content if the operation succeeds.
func (controller deleteController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	accountID := path.Base(req.Path)
	deletedAccount, err := controller.Dao.DeleteAccount(accountID)

	// Handle DB errors
	if err != nil {
		switch err.(type) {
		case *db.AccountNotFoundError:
			return response.NotFoundError(), nil
		case *db.AccountLeasedError:
			return response.CreateAPIErrorResponse(http.StatusConflict, response.CreateErrorResponse("Conflict", err.Error())), nil
		default:
			return response.CreateAPIErrorResponse(http.StatusInternalServerError, response.CreateErrorResponse("ServerError", "Internal Server Error")), nil
		}
	}

	// Delete the IAM Principal Role for the account
	controller.destroyIAMPrincipal(deletedAccount)

	// Publish SNS "account-deleted" message
	controller.sendSNS(deletedAccount)

	// Push the account to the Reset Queue, so it gets cleaned up
	controller.sendToResetQueue(deletedAccount.ID)

	return response.CreateAPIResponse(http.StatusNoContent, ""), nil
}

// sendSNS sends notification to SNS that the delete has occurred.
func (controller deleteController) sendSNS(account *db.DceAccount) {
	serializedAccount := response.AccountResponse(*account)
	serializedMessage, err := common.PrepareSNSMessageJSON(serializedAccount)

	if err != nil {
		log.Printf("Failed to serialized SNS message for account %s: %s", account.ID, err)
		return
	}

	_, err = controller.SNS.PublishMessage(&controller.AccountDeletedTopicArn, &serializedMessage, true)
	if err != nil {
		log.Printf("Failed to publish SNS message for account %s: %s", account.ID, err)
	}
}

// sendToResetQueue sends the account to the reset queue
func (controller deleteController) sendToResetQueue(accountID string) {
	err := controller.Queue.SendMessage(&controller.ResetQueueURL, &accountID)
	if err != nil {
		log.Printf("Failed to add account %s to reset Queue: %s", accountID, err)
	}
}

func (controller deleteController) destroyIAMPrincipal(account *db.DceAccount) {
	// Assume role into the new Dce account
	accountSession, err := controller.TokenService.NewSession(&controller.AWSSession, account.AdminRoleArn)
	if err != nil {
		log.Printf("Failed to assume role into account %s: %s", account.ID, err)
		return
	}
	iamClient := iam.New(accountSession)

	// Destroy the role and policy
	controller.RoleManager.SetIAMClient(iamClient)
	_, err = controller.RoleManager.DestroyRoleWithPolicy(&rolemanager.DestroyRoleWithPolicyInput{
		RoleName:  controller.PrincipalRoleName,
		PolicyArn: fmt.Sprintf("arn:aws:iam::%s:policy/%s", account.ID, controller.PrincipalPolicyName),
	})
	// Log error, and continue
	if err != nil {
		log.Printf("Failed to destroy Dce Principal IAM Role and Policy: %s", err)
	}
}
