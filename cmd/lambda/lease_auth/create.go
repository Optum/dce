package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/api/response"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
)

// CreateController - Create Lease Authentication Information
type CreateController struct {
	Dao           db.DBer
	TokenService  common.TokenService
	ConsoleURL    string
	FederationURL string
	UserDetailer  api.UserDetailer
}

// Call - function to return a specific AWS Lease record to the request
func (controller CreateController) Call(ctx context.Context, req *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	leaseID := req.PathParameters["id"]

	// Get the Lease Information
	lease, err := controller.Dao.GetLeaseByID(leaseID)
	if err != nil {
		log.Printf("Error Getting Lease (%s) by Id: %s", leaseID, err)
		return response.CreateAPIGatewayErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Failed Get on Lease %s",
					leaseID))), nil
	}
	if lease == nil {
		log.Printf("Error Getting Lease (%s) by Id: %s", leaseID, err)
		return response.NotFoundError(), nil
	}
	// Don't return any lease information if the lease isn't active
	if lease.LeaseStatus != db.Active {
		log.Printf("Lease (%s) isn't in an active state", leaseID)
		return response.UnauthorizedError(), nil
	}

	// Get the User Information
	user := controller.UserDetailer.GetUser(req)
	if user.Role != api.AdminGroupName {
		if lease.PrincipalID != user.Username {
			log.Printf("User (%s) doesn't have access to lease %s", user.Username, leaseID)
			return response.NotFoundError(), nil
		}
	}

	// Get the Account Information
	accountID := lease.AccountID
	account, err := controller.Dao.GetAccount(accountID)
	if err != nil {
		log.Printf("Error Getting Account (%s) by Id: %s", accountID, err)
		return response.CreateAPIGatewayErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Failed List on Account %s", accountID))), nil
	}
	if account == nil {
		log.Printf("Account (%s) doesn't exist", accountID)
		return response.CreateAPIGatewayErrorResponse(http.StatusInternalServerError,
			response.CreateErrorResponse("ServerError",
				fmt.Sprintf("Account %s could not be found", accountID))), nil
	}

	log.Printf("Assuming Role: %s", account.PrincipalRoleArn)
	roleSessionName := user.Username
	if roleSessionName == "" {
		roleSessionName = lease.PrincipalID
	}
	assumeRoleInputs := sts.AssumeRoleInput{
		RoleArn:         &account.PrincipalRoleArn,
		RoleSessionName: aws.String(roleSessionName),
	}
	assumeRoleOutput, err := controller.TokenService.AssumeRole(
		&assumeRoleInputs,
	)
	if err != nil {
		log.Printf("Failed to assume role %s: %s", *assumeRoleInputs.RoleArn, err.Error())
		return response.ServerError(), nil
	}

	consoleURL, err := controller.buildConsoleURL(*assumeRoleOutput.Credentials)
	if err != nil {
		log.Printf("Error building signin url: %s", err)
		return response.ServerError(), nil
	}
	result := response.LeaseAuthResponse{
		AccessKeyID:     *assumeRoleOutput.Credentials.AccessKeyId,
		SecretAccessKey: *assumeRoleOutput.Credentials.SecretAccessKey,
		SessionToken:    *assumeRoleOutput.Credentials.SessionToken,
		ConsoleURL:      consoleURL,
	}
	return response.CreateAPIGatewayJSONResponse(http.StatusCreated, result), nil
}

func (controller CreateController) buildConsoleURL(creds sts.Credentials) (string, error) {

	signinToken, err := controller.getSigninToken(creds)
	if err != nil {
		log.Printf("Error when getting signin token: %s", err)
		return "", err
	}

	// have to use url.QueryEscape for the URL or its not properly escaped
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s?Destination=%s", controller.FederationURL, url.QueryEscape(controller.ConsoleURL)),
		nil)
	if err != nil {
		log.Printf("Error building request: %s", err)
		return "", err
	}
	q := req.URL.Query()
	q.Add("Action", "login")
	q.Add("Issuer", "DCE")
	q.Add("SigninToken", signinToken)
	req.URL.RawQuery = q.Encode()

	return req.URL.String(), nil
}

func (controller CreateController) getSigninToken(creds sts.Credentials) (string, error) {
	type signinCredentialsInput struct {
		AccessKeyID     string `json:"sessionId"`
		SecretAccessKey string `json:"sessionKey"`
		SessionToken    string `json:"sessionToken"`
	}
	credentialString, err := json.Marshal(&signinCredentialsInput{
		AccessKeyID:     *creds.AccessKeyId,
		SecretAccessKey: *creds.SecretAccessKey,
		SessionToken:    *creds.SessionToken,
	})
	if err != nil {
		log.Printf("Error marshalling credentials: %s", err)
		return "", err
	}

	req, err := http.NewRequest("GET", controller.FederationURL, nil)
	if err != nil {
		log.Printf("Error building request for signin token: %s", err)
		return "", err
	}
	q := req.URL.Query()
	q.Add("Action", "getSigninToken")
	q.Add("Session", string(credentialString))
	req.URL.RawQuery = q.Encode()

	httpClient := http.Client{}
	type SigninToken struct {
		SigninToken string `json:"SigninToken"`
	}
	resSigninToken, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error getting signing token: %s", err)
		return "", err
	}
	bodySigninToken, err := ioutil.ReadAll(resSigninToken.Body)
	if err != nil {
		log.Printf("Error getting signing token: %s", err)
		return "", err
	}

	var signinToken SigninToken
	json.Unmarshal(bodySigninToken, &signinToken)
	return signinToken.SigninToken, nil
}
