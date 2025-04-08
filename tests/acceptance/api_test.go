package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/codebuild/codebuildiface"
	"github.com/aws/aws-sdk-go/service/cognitoidentity"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"

	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/aws/client"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	sigv4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-sdk-go/service/iam"
	aws2 "github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"

	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/usage"
	"github.com/Optum/dce/tests/testutils"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

var (
	dbSvc              *db.DB
	usageSvc           *usage.DB
	sqsSvc             sqsiface.SQSAPI
	codeBuildSvc       codebuildiface.CodeBuildAPI
	sqsResetURL        string
	codeBuildResetName string
)

func TestApi(t *testing.T) {
	// Grab the API url from Terraform output
	tfOpts := &terraform.Options{
		TerraformDir: "../../modules",
	}
	tfOut := terraform.OutputAll(t, tfOpts)

	apiURL, ok := tfOut["api_url"].(string)
	if !ok || apiURL == "" {
		t.Fatalf("api_url should be a non-empty string, but got: %v", tfOut["api_url"])
	}
	require.NotEmpty(t, apiURL, "api_url should not be empty")

	awsRegion, ok := tfOut["aws_region"].(string)
	if !ok || awsRegion == "" {
		t.Fatalf("aws_region should be a non-empty string, but got: %v", tfOut["aws_region"])
	}
	require.NotEmpty(t, awsRegion, "aws_region should not be empty")

	accountsTableName, ok := tfOut["accounts_table_name"].(string)
	if !ok || accountsTableName == "" {
		t.Fatalf("accounts_table_name should be a non-empty string, but got: %v", tfOut["accounts_table_name"])
	}
	require.NotEmpty(t, accountsTableName, "accounts_table_name should not be empty")

	leasesTableName, ok := tfOut["leases_table_name"].(string)
	if !ok || leasesTableName == "" {
		t.Fatalf("leases_table_name should be a non-empty string, but got: %v", tfOut["leases_table_name"])
	}
	require.NotEmpty(t, leasesTableName, "leases_table_name should not be empty")

	// Configure the DB service
	awsSession, err := session.NewSession()
	require.Nil(t, err)
	dbSvc = db.New(
		dynamodb.New(
			awsSession,
			aws.NewConfig().WithRegion(awsRegion),
		),
		accountsTableName,
		leasesTableName,
		7,
	)
}

type leaseRequest struct {
	PrincipalID string `json:"principalId"`
	AccountID   string `json:"accountId"`
}

type inputLeaseRequest struct {
	PrincipalID  string  `json:"principalId"`
	AccountID    string  `json:"accountId"`
	BudgetAmount float64 `json:"budgetAmount"`
	ExpiresOn    int64   `json:"expiresOn"`
}

type createAccountRequest struct {
	ID           string `json:"id"`
	AdminRoleArn string `json:"adminRoleArn"`
}

type apiRequestInput struct {
	method      string
	url         string
	creds       *credentials.Credentials
	region      string
	json        interface{}
	maxAttempts int
	// Callback function to assert API responses.
	// apiRequest() will continue to retry until this
	// function passes assertions.
	//
	// eg.
	//		f: func(r *testutils.R, apiResp *apiResponse) {
	//			assert.Equal(r, 200, apiResp.StatusCode)
	//		},
	// or:
	//		f: statusCodeAssertion(200)
	//
	// By default, this will check that the API returns a 2XX response
	f func(r *testutils.R, apiResp *apiResponse)
}

func statusCodeAssertion(statusCode int) func(r *testutils.R, apiResp *apiResponse) {
	return func(r *testutils.R, apiResp *apiResponse) {
		// Defaults to returning 200
		assert.Equal(r, statusCode, apiResp.StatusCode)
	}
}

type apiResponse struct {
	http.Response
	json interface{}
}

var chainCredentials = credentials.NewChainCredentials([]credentials.Provider{
	&credentials.EnvProvider{},
	&credentials.SharedCredentialsProvider{Filename: "", Profile: ""},
})

func apiRequest(t *testing.T, input *apiRequestInput) *apiResponse {
	// Set defaults
	if input.creds == nil {
		input.creds = chainCredentials
	}
	if input.region == "" {
		input.region = "us-east-1"
	}
	if input.maxAttempts == 0 {
		input.maxAttempts = 30
	}

	// Create API request
	req, err := http.NewRequest(input.method, input.url, nil)
	assert.Nil(t, err)

	// Sign our API request, using sigv4
	// See https://docs.aws.amazon.com/general/latest/gr/sigv4_signing.html
	signer := sigv4.NewSigner(input.creds)
	now := time.Now().Add(time.Duration(30) * time.Second)
	var signedHeaders http.Header
	var apiResp *apiResponse
	testutils.Retry(t, input.maxAttempts, 2*time.Second, func(r *testutils.R) {
		// If there's a json provided, add it when signing
		// Body does not matter if added before the signing, it will be overwritten
		if input.json != nil {
			payload, err := json.Marshal(input.json)
			assert.Nil(t, err)
			req.Header.Set("Content-Type", "application/json")
			signedHeaders, err = signer.Sign(req, bytes.NewReader(payload),
				"execute-api", input.region, now)
			require.Nil(t, err)
		} else {
			signedHeaders, err = signer.Sign(req, nil, "execute-api",
				input.region, now)
		}
		assert.NoError(r, err)
		assert.NotNil(r, signedHeaders)

		// Send the API requests
		// resp, err := http.DefaultClient.Do(req)
		httpClient := http.Client{
			Timeout: 60 * time.Second,
		}
		resp, err := httpClient.Do(req)
		assert.NoError(r, err)
		assert.NotNil(r, resp)

		if resp != nil {
			// Parse the JSON response
			apiResp = &apiResponse{
				Response: *resp,
			}
			defer resp.Body.Close()
			var data interface{}

			body, err := io.ReadAll(resp.Body)
			assert.NoError(r, err)

			err = json.Unmarshal([]byte(body), &data)
			if err == nil {
				apiResp.json = data
			}

			if input.f != nil {
				input.f(r, apiResp)
			}
		}

	})
	return apiResp
}

func parseResponseJSON(t require.TestingT, resp *apiResponse) map[string]interface{} {
	require.NotNil(t, resp.json)
	return resp.json.(map[string]interface{})
}

func responseJSONString(t require.TestingT, resp *apiResponse, key string) string {
	resJSON := parseResponseJSON(t, resp)
	val, ok := resJSON[key]
	assert.True(t, ok, "response has key %s", key)
	valStr, ok := val.(string)
	assert.True(t, ok, "response key %s is string: %v", key, val)
	return valStr
}

func parseResponseArrayJSON(t require.TestingT, resp *apiResponse) []map[string]interface{} {
	require.NotNil(t, resp.json)

	// Go doesn't allow you to cast directly to []map[string]interface{}
	// so we need to mess around here a bit.
	// This might be relevant: https://stackoverflow.com/questions/38579485/golang-convert-slices-into-map
	require.IsTypef(t, []interface{}{}, resp.json, "Expected JSON array response, got %v", resp.json)
	respJSON := resp.json.([]interface{})

	arrJSON := []map[string]interface{}{}
	for _, obj := range respJSON {
		arrJSON = append(arrJSON, obj.(map[string]interface{}))
	}

	return arrJSON
}

func createPolicy(t *testing.T, awsSession client.ConfigProvider, name string, body string) *iam.Policy {
	iamSvc := iam.New(awsSession)
	policy, err := iamSvc.CreatePolicy(&iam.CreatePolicyInput{
		PolicyDocument: &body,
		PolicyName:     &name,
	})

	// Ignore errors indicating the policy already exists (e.g. if a previous test run already created the policy)
	if err != nil && strings.Contains(err.Error(), iam.ErrCodeEntityAlreadyExistsException) {
		err = nil
	}
	require.Nil(t, err)
	return policy.Policy
}

type createAdminRoleOutput struct {
	accountID    string
	roleName     string
	adminRoleArn string
}

func createAdminRole(t *testing.T, awsSession client.ConfigProvider, adminRoleName string, policies []string) *createAdminRoleOutput {
	currentAccountID := aws2.GetAccountId(t)

	// Create an Admin Role that can be assumed
	// within this account
	iamSvc := iam.New(awsSession)
	assumeRolePolicy := fmt.Sprintf(`{
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
		}`, currentAccountID)
	roleRes, err := iamSvc.CreateRole(&iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(assumeRolePolicy),
		Path:                     aws.String("/"),
		RoleName:                 aws.String(adminRoleName),
	})
	require.Nil(t, err)

	adminRoleArn := *roleRes.Role.Arn

	for _, p := range policies {
		_, err = iamSvc.AttachRolePolicy(&iam.AttachRolePolicyInput{
			RoleName:  aws.String(adminRoleName),
			PolicyArn: aws.String(p),
		})
		require.Nil(t, err)
	}

	// Wait for the role to be assumable
	log.Println("Created admin test role. Waiting for role to be assumeable")
	testutils.Retry(t, 30, time.Second, func(r *testutils.R) {
		// This might take a bit.
		// Log progress, so we know our tests aren't stuck
		if r.Attempt == 1 || r.Attempt%5 == 0 {
			log.Printf("Waiting for admin role to be assumeable: %s", adminRoleArn)
		}

		creds := stscreds.NewCredentials(awsSession, adminRoleArn)
		_, err := creds.Get()
		assert.Nilf(r, err, "Unable to assume admin test role: %s", err)
	})

	return &createAdminRoleOutput{
		adminRoleArn: adminRoleArn,
		roleName:     adminRoleName,
		accountID:    currentAccountID,
	}
}

func createUsage(t *testing.T, apiURL string, usageSvc usage.DBer) {
	// Create usage
	// Setup usage dates
	const ttl int = 3
	currentDate := time.Now()
	testStartDate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, time.UTC)
	testEndDate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 23, 59, 59, 59, time.UTC)

	usageStartDate := testStartDate
	usageEndDate := testEndDate
	startDate := testStartDate
	endDate := testEndDate

	timeToLive := startDate.AddDate(0, 0, ttl)

	var testPrincipalID = "TestUser1"
	var testAccountID = "123456789012"

	for i := 1; i <= 5; i++ {

		input, err := usage.NewUsage(
			usage.NewUsageInput{
				PrincipalID:  testPrincipalID,
				AccountID:    testAccountID,
				StartDate:    startDate.Unix(),
				EndDate:      endDate.Unix(),
				CostAmount:   2000.00,
				CostCurrency: "USD",
				TimeToLive:   timeToLive.Unix(),
			},
		)
		require.Nil(t, err)
		err = usageSvc.PutUsage(*input)
		require.Nil(t, err)

		usageEndDate = endDate
		startDate = startDate.AddDate(0, 0, -1)
		endDate = endDate.AddDate(0, 0, -1)
	}

	queryString := fmt.Sprintf("/usage?startDate=%d&endDate=%d", usageStartDate.Unix(), usageEndDate.Unix())

	testutils.Retry(t, 10, 10*time.Millisecond, func(r *testutils.R) {

		resp := apiRequest(t, &apiRequestInput{
			method: "GET",
			url:    apiURL + queryString,
			json:   nil,
		})

		// Verify response code
		assert.Equal(r, http.StatusOK, resp.StatusCode)

		// Parse response json
		data := parseResponseArrayJSON(t, resp)

		//Verify response json
		if len(data) > 0 && data[0] != nil {
			usageJSON := data[0]
			assert.Equal(r, "TestUser1", usageJSON["principalId"].(string))
			assert.Equal(r, "TestAcct1", usageJSON["accountId"].(string))
			assert.Equal(r, 10000.00, usageJSON["costAmount"].(float64))
		}
	})
}

func NewCredentials(t *testing.T, awsSession *session.Session, roleArn string) *credentials.Credentials {

	var creds *credentials.Credentials
	testutils.Retry(t, 10, 2*time.Second, func(r *testutils.R) {

		creds = stscreds.NewCredentials(awsSession, roleArn)
		assert.NotNil(r, creds)
	})
	return creds
}

func deleteAdminRole(t *testing.T, role string, policies []string) {
	awsSession, _ := session.NewSession()
	iamSvc := iam.New(awsSession)
	testutils.Retry(t, 10, 2*time.Second, func(r *testutils.R) {
		for _, p := range policies {
			_, err := iamSvc.DetachRolePolicy(&iam.DetachRolePolicyInput{
				RoleName:  aws.String(role),
				PolicyArn: aws.String(p),
			})
			assert.Nil(t, err)
		}
		_, err := iamSvc.DeleteRole(&iam.DeleteRoleInput{
			RoleName: aws.String(role),
		})
		assert.Nil(t, err)
	})
}

func deletePolicy(t *testing.T, policyArn string) {
	awsSession, _ := session.NewSession()
	iamSvc := iam.New(awsSession)
	testutils.Retry(t, 10, 2*time.Second, func(r *testutils.R) {
		_, err := iamSvc.DeletePolicy(&iam.DeletePolicyInput{
			PolicyArn: aws.String(policyArn),
		})
		assert.Nil(t, err)
	})
}

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
func getRandString(t *testing.T, n int, letters string) string {
	t.Helper()
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Int63()%int64(len(letters))]
	}
	return string(b)
}

type CognitoUser struct {
	UserCredsValue credentials.Value
	Username       string
	UserPoolID     string
}

func (u CognitoUser) delete(t *testing.T, tfOut map[string]interface{}, adminSession *session.Session) {
	userPoolSvc := cognitoidentityprovider.New(
		adminSession,
		aws.NewConfig().WithRegion(tfOut["aws_region"].(string)),
	)

	_, err := userPoolSvc.AdminDeleteUser(&cognitoidentityprovider.AdminDeleteUserInput{
		UserPoolId: &u.UserPoolID,
		Username:   &u.Username,
	})
	assert.Nil(t, err)
}
func NewCognitoUser(t *testing.T, tfOut map[string]interface{}, awsSession *session.Session, accountID string) CognitoUser {
	cognitoUser := CognitoUser{}

	userPoolSvc := cognitoidentityprovider.New(
		awsSession,
		aws.NewConfig().WithRegion(tfOut["aws_region"].(string)),
	)

	identityPoolSvc := cognitoidentity.New(
		awsSession,
		aws.NewConfig().WithRegion(tfOut["aws_region"].(string)),
	)
	// Create user
	cognitoUser.Username = getRandString(t, 8, "abcdefghijklmnopqrstuvwxyz")
	tempPassword := getRandString(t, 4, "abcdefghijklmnopqrstuvwxyz") +
		getRandString(t, 2, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") +
		getRandString(t, 2, "123456789") +
		getRandString(t, 1, "!^*")

	supress := "SUPPRESS"
	cognitoUser.UserPoolID = tfOut["cognito_user_pool_id"].(string)
	_, err := userPoolSvc.AdminCreateUser(&cognitoidentityprovider.AdminCreateUserInput{
		MessageAction:     &supress,
		TemporaryPassword: &tempPassword,
		UserPoolId:        &cognitoUser.UserPoolID,
		Username:          &cognitoUser.Username,
	})
	if err != nil {
		defer cognitoUser.delete(t, tfOut, awsSession)
	}
	require.Nil(t, err)

	// Reset user's password
	permPassword := getRandString(t, 4, "abcdefghijklmnopqrstuvwxyz") +
		getRandString(t, 2, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") +
		getRandString(t, 2, "123456789") +
		getRandString(t, 1, "!^*")
	permanent := true
	_, err = userPoolSvc.AdminSetUserPassword(&cognitoidentityprovider.AdminSetUserPasswordInput{
		Password:   &permPassword,
		Permanent:  &permanent,
		UserPoolId: &cognitoUser.UserPoolID,
		Username:   &cognitoUser.Username,
	})
	if err != nil {
		defer cognitoUser.delete(t, tfOut, awsSession)
	}
	require.Nil(t, err)

	// Update user pool client to allow ADMIN_USER_PASSWORD_AUTH
	clientID := tfOut["cognito_user_pool_client_id"].(string)
	describeUserPoolClientOutput, err := userPoolSvc.DescribeUserPoolClient(&cognitoidentityprovider.DescribeUserPoolClientInput{
		ClientId:   &clientID,
		UserPoolId: &cognitoUser.UserPoolID,
	})
	if err != nil {
		defer cognitoUser.delete(t, tfOut, awsSession)
	}
	require.Nil(t, err)
	ALLOW_REFRESH_TOKEN_AUTH := "ALLOW_REFRESH_TOKEN_AUTH"
	ALLOW_ADMIN_USER_PASSWORD_AUTH := "ALLOW_ADMIN_USER_PASSWORD_AUTH"
	allowedAuthFlows := []*string{&ALLOW_REFRESH_TOKEN_AUTH, &ALLOW_ADMIN_USER_PASSWORD_AUTH}
	_, err = userPoolSvc.UpdateUserPoolClient(&cognitoidentityprovider.UpdateUserPoolClientInput{
		ClientId:          &clientID,
		ExplicitAuthFlows: allowedAuthFlows,
		UserPoolId:        &cognitoUser.UserPoolID,
		CallbackURLs:      describeUserPoolClientOutput.UserPoolClient.CallbackURLs,
		LogoutURLs:        describeUserPoolClientOutput.UserPoolClient.LogoutURLs,
	})
	if err != nil {
		defer cognitoUser.delete(t, tfOut, awsSession)
	}
	require.Nil(t, err)

	// authenticate with use pool to get Access, Identity, and Refresh JWTs
	userCreds := make(map[string]*string)
	userCreds["USERNAME"] = &cognitoUser.Username
	userCreds["PASSWORD"] = &permPassword
	adminAuthFlow := "ADMIN_USER_PASSWORD_AUTH"
	output, err := userPoolSvc.AdminInitiateAuth(&cognitoidentityprovider.AdminInitiateAuthInput{
		AuthFlow:       &adminAuthFlow,
		AuthParameters: userCreds,
		ClientId:       &clientID,
		UserPoolId:     &cognitoUser.UserPoolID,
	})
	if err != nil {
		defer cognitoUser.delete(t, tfOut, awsSession)
	}
	require.Nil(t, err)

	// Exchange Identity JWT with identity pool for iam creds
	// https://github.com/aws/aws-sdk-go/issues/406#issuecomment-150666885
	userPoolProviderName := tfOut["cognito_user_pool_endpoint"].(string)
	identityPoolID := tfOut["cognito_identity_pool_id"].(string)
	var logins = make(map[string]*string)
	logins[userPoolProviderName] = output.AuthenticationResult.IdToken
	identityID, err := identityPoolSvc.GetId(&cognitoidentity.GetIdInput{
		AccountId:      &accountID,
		IdentityPoolId: &identityPoolID,
		Logins:         logins,
	})
	if err != nil {
		defer cognitoUser.delete(t, tfOut, awsSession)
	}
	require.Nil(t, err)

	idCredOutput, err := identityPoolSvc.GetCredentialsForIdentity(&cognitoidentity.GetCredentialsForIdentityInput{
		IdentityId: identityID.IdentityId,
		Logins:     logins,
	})
	if err != nil {
		defer cognitoUser.delete(t, tfOut, awsSession)
	}
	require.Nil(t, err)

	// Change session to use user creds
	cognitoUser.UserCredsValue = credentials.Value{
		AccessKeyID:     *idCredOutput.Credentials.AccessKeyId,
		SecretAccessKey: *idCredOutput.Credentials.SecretKey,
		SessionToken:    *idCredOutput.Credentials.SessionToken,
	}

	return cognitoUser
}

func waitForAccountStatus(t *testing.T, apiURL, accountID, expectedStatus string) *apiResponse {
	res := apiRequest(t, &apiRequestInput{
		method:      "GET",
		url:         apiURL + "/accounts/" + accountID,
		maxAttempts: 240,
		f: func(r *testutils.R, res *apiResponse) {
			assert.Equalf(r, 200, res.StatusCode, "%v", res.json)

			actualStatus := responseJSONString(t, res, "accountStatus")

			// These status changes can take a while. Log output,
			// so we know our tests aren't stuck
			if r.Attempt == 1 || r.Attempt%5 == 0 {
				log.Printf("Waiting for account to be %s in test %q. Account is %s", expectedStatus, t.Name(), actualStatus)
			}
			assert.Equalf(r, expectedStatus, actualStatus,
				"Expected account status to change to %s in test %q", expectedStatus, t.Name())
		},
	})

	// Fail now if the status change never happened
	actualStatus := responseJSONString(t, res, "accountStatus")
	require.Equalf(t, expectedStatus, actualStatus,
		"Expected account status to change from %s to %s", actualStatus, expectedStatus)

	time.Sleep(time.Second * 5)

	return res
}
