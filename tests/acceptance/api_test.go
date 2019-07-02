package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Optum/Redbox/pkg/api/response"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	sigv4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-sdk-go/service/iam"
	aws2 "github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"

	"github.com/Optum/Redbox/pkg/db"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func TestApi(t *testing.T) {
	// Grab the API url from Terraform output
	tfOpts := &terraform.Options{
		TerraformDir: "../../modules",
	}
	tfOut := terraform.OutputAll(t, tfOpts)
	apiURL := tfOut["api_url"].(string)
	require.NotNil(t, apiURL)

	// Configure the DB service
	awsSession, err := session.NewSession()
	require.Nil(t, err)
	dbSvc := db.New(
		dynamodb.New(
			awsSession,
			aws.NewConfig().WithRegion(tfOut["aws_region"].(string)),
		),
		tfOut["dynamodb_table_account_name"].(string),
		tfOut["dynamodb_table_account_assignment_name"].(string),
	)

	// Cleanup tables, to start out
	truncateAccountTable(t, dbSvc)
	truncateAccountAssignmentTable(t, dbSvc)

	t.Run("Authentication", func(t *testing.T) {

		t.Run("should forbid unauthenticated requests", func(t *testing.T) {
			// Send request to the /status API
			resp, err := http.Get(apiURL + "/leases")
			require.Nil(t, err)

			// Should receive a 403
			require.Equal(t, http.StatusForbidden, resp.StatusCode,
				"should return a 403")

			// Parse response json
			defer resp.Body.Close()
			var data map[string]string
			err = json.NewDecoder(resp.Body).Decode(&data)
			require.Nil(t, err)

			// Should return an Auth error message
			require.Equal(t, "Missing Authentication Token", data["message"])
		})

		t.Run("should allow IAM authenticated requests", func(t *testing.T) {
			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases",
			})

			// Defaults to returning 200
			require.Equal(t, http.StatusOK, resp.StatusCode)

			// Parse response json
			data := parseResponseJSON(t, resp)

			// Currently returns a default message
			require.Equal(t, "pong", data["message"].(string))
		})

	})

	t.Run("api_execute_admin policy", func(t *testing.T) {

		t.Run("should allow executing Redbox APIs", func(t *testing.T) {
			// Don't run this test, if using `go test -short` flag
			if testing.Short() {
				t.Skip("Skipping tests in short mode. IAM role takes a while to propagate...")
			}

			// Grab policy name from Terraform outputs
			policyArn := terraform.Output(t, tfOpts, "api_access_policy_arn")
			require.NotNil(t, policyArn)

			// Configure IAM service
			awsSession, err := session.NewSession()
			require.Nil(t, err)
			iamSvc := iam.New(awsSession)

			// Create a Role we can assume, to test out our policy
			accountID := aws2.GetAccountId(t)
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
}`, accountID)
			roleName := "redbox-api-execute-test-role-" + fmt.Sprintf("%v", time.Now().Unix())
			roleRes, err := iamSvc.CreateRole(&iam.CreateRoleInput{
				AssumeRolePolicyDocument: aws.String(assumeRolePolicy),
				Path:                     aws.String("/"),
				RoleName:                 aws.String(roleName),
			})
			require.Nil(t, err)

			// Cleanup: Delete the Role
			defer func() {
				_, err = iamSvc.DeleteRole(&iam.DeleteRoleInput{
					RoleName: aws.String(roleName),
				})
				require.Nil(t, err)
			}()

			// Attach our managed API access policy to the roleRes
			_, err = iamSvc.AttachRolePolicy(&iam.AttachRolePolicyInput{
				PolicyArn: aws.String(policyArn),
				RoleName:  aws.String(roleName),
			})
			require.Nil(t, err)

			// Cleanup: Detach the policy from the role (required to delete the Role)
			defer func() {
				_, err = iamSvc.DetachRolePolicy(&iam.DetachRolePolicyInput{
					PolicyArn: aws.String(policyArn),
					RoleName:  aws.String(roleName),
				})
				require.Nil(t, err)
			}()

			// IAM Role takes a while to propagate....
			time.Sleep(10 * time.Second)

			// Assume the roleRes we just created
			roleCreds := stscreds.NewCredentials(awsSession, *roleRes.Role.Arn)

			// Attempt to hit the API with using our assumed role
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases",
				creds:  roleCreds,
			})

			require.NotEqual(t, http.StatusForbidden, resp.StatusCode,
				"Should not return an IAM authorization error")
		})

	})

	t.Run("Provisioning and Decommissioning", func(t *testing.T) {

		t.Run("Should be able to provision and decommission", func(t *testing.T) {
			defer truncateAccountTable(t, dbSvc)
			defer truncateAccountAssignmentTable(t, dbSvc)

			// Create an Account Entry
			acctID := "123"
			userID := "user"
			timeNow := time.Now().Unix()
			err := dbSvc.PutAccount(db.RedboxAccount{
				ID:             acctID,
				AccountStatus:  db.Ready,
				LastModifiedOn: timeNow,
			})
			require.Nil(t, err)

			// Create the Provision Request Body
			body := leaseRequest{
				UserID: userID,
			}

			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "POST",
				url:    apiURL + "/leases",
				json:   body,
			})

			// Verify response code
			require.Equal(t, http.StatusCreated, resp.StatusCode)

			// Parse response json
			data := parseResponseJSON(t, resp)

			// Verify provisioned response json
			require.Equal(t, userID, data["userId"].(string))
			require.Equal(t, acctID, data["accountId"].(string))
			require.Equal(t, string(db.Active),
				data["assignmentStatus"].(string))
			require.NotNil(t, data["createdOn"])
			require.NotNil(t, data["lastModifiedOn"])

			// Create the Decommission Request Body
			body = leaseRequest{
				UserID:    userID,
				AccountID: acctID,
			}

			// Send an API request
			resp = apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + "/leases",
				json:   body,
			})

			// Verify response code
			require.Equal(t, http.StatusOK, resp.StatusCode)

			// Parse response json
			data = parseResponseJSON(t, resp)

			// Verify provisioned response json
			require.Equal(t, userID, data["userId"].(string))
			require.Equal(t, acctID, data["accountId"].(string))
			require.Equal(t, string(db.Decommissioned),
				data["assignmentStatus"].(string))
			require.NotNil(t, data["createdOn"])
			require.NotNil(t, data["lastModifiedOn"])

		})

		t.Run("Should not be able to provision with empty json", func(t *testing.T) {
			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "POST",
				url:    apiURL + "/leases",
			})

			// Verify response code
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// Parse response json
			data := parseResponseJSON(t, resp)

			// Verify error response json
			// Get nested json in response json
			err := data["error"].(map[string]interface{})
			require.Equal(t, "ClientError", err["code"].(string))
			require.Equal(t, "Failed to Parse Request Body: ",
				err["message"].(string))
		})

		t.Run("Should not be able to provision with no available accounts", func(t *testing.T) {
			// Create the Provision Request Body
			userID := "user"
			body := leaseRequest{
				UserID: userID,
			}

			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "POST",
				url:    apiURL + "/leases",
				json:   body,
			})

			// Verify response code
			require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

			// Parse response json
			data := parseResponseJSON(t, resp)

			// Verify error response json
			// Get nested json in response json
			err := data["error"].(map[string]interface{})
			require.Equal(t, "ServerError", err["code"].(string))
			require.Equal(t, "No Available Redbox Accounts at this moment",
				err["message"].(string))
		})

		t.Run("Should not be able to provision with an existing account", func(t *testing.T) {
			defer truncateAccountTable(t, dbSvc)
			defer truncateAccountAssignmentTable(t, dbSvc)

			// Create an Account Entry
			acctID := "123"
			userID := "user"
			timeNow := time.Now().Unix()
			err := dbSvc.PutAccount(db.RedboxAccount{
				ID:             acctID,
				AccountStatus:  db.Assigned,
				LastModifiedOn: timeNow,
			})
			require.Nil(t, err)

			// Create an Assignment Entry
			_, err = dbSvc.PutAccountAssignment(db.RedboxAccountAssignment{
				UserID:           userID,
				AccountID:        acctID,
				AssignmentStatus: db.Active,
				CreatedOn:        timeNow,
				LastModifiedOn:   timeNow,
			})
			require.Nil(t, err)

			// Create the Provision Request Body
			body := leaseRequest{
				UserID: userID,
			}

			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "POST",
				url:    apiURL + "/leases",
				json:   body,
			})

			// Verify response code
			require.Equal(t, http.StatusConflict, resp.StatusCode)

			// Parse response json
			data := parseResponseJSON(t, resp)

			// Verify error response json
			// Get nested json in response json
			errResp := data["error"].(map[string]interface{})
			require.Equal(t, "ClientError", errResp["code"].(string))
			require.Equal(t, "User already has an existing Redbox: 123",
				errResp["message"].(string))
		})

		t.Run("Should not be able to decommission with empty json", func(t *testing.T) {
			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + "/leases",
			})

			// Verify response code
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// Parse response json
			data := parseResponseJSON(t, resp)

			// Verify error response json
			// Get nested json in response json
			err := data["error"].(map[string]interface{})
			require.Equal(t, "ClientError", err["code"].(string))
			require.Equal(t, "Failed to Parse Request Body: ",
				err["message"].(string))
		})

		t.Run("Should not be able to decommission with no assignments", func(t *testing.T) {
			// Create the Provision Request Body
			userID := "user"
			acctID := "123"
			body := leaseRequest{
				UserID:    userID,
				AccountID: acctID,
			}

			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + "/leases",
				json:   body,
			})

			// Verify response code
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// Parse response json
			data := parseResponseJSON(t, resp)

			// Verify error response json
			// Get nested json in response json
			err := data["error"].(map[string]interface{})
			require.Equal(t, "ClientError", err["code"].(string))
			require.Equal(t, "No account assignments found for user",
				err["message"].(string))
		})

		t.Run("Should not be able to decommission with wrong account", func(t *testing.T) {
			// Create an Account Entry
			acctID := "123"
			userID := "user"
			timeNow := time.Now().Unix()
			err := dbSvc.PutAccount(db.RedboxAccount{
				ID:             acctID,
				AccountStatus:  db.Assigned,
				LastModifiedOn: timeNow,
			})
			require.Nil(t, err)

			// Create an Assignment Entry
			_, err = dbSvc.PutAccountAssignment(db.RedboxAccountAssignment{
				UserID:           userID,
				AccountID:        acctID,
				AssignmentStatus: db.Active,
				CreatedOn:        timeNow,
				LastModifiedOn:   timeNow,
			})
			require.Nil(t, err)

			// Create the Provision Request Body
			wrongAcctID := "456"
			body := leaseRequest{
				UserID:    userID,
				AccountID: wrongAcctID,
			}

			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + "/leases",
				json:   body,
			})

			// Verify response code
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// Parse response json
			data := parseResponseJSON(t, resp)

			// Verify error response json
			// Get nested json in response json
			errResp := data["error"].(map[string]interface{})
			require.Equal(t, "ClientError", errResp["code"].(string))
			require.Equal(t, "No active account assignments found for user",
				errResp["message"].(string))
		})

		t.Run("Should not be able to decommission with decommissioned account", func(t *testing.T) {
			// Create an Account Entry
			acctID := "123"
			userID := "user"
			timeNow := time.Now().Unix()
			err := dbSvc.PutAccount(db.RedboxAccount{
				ID:             acctID,
				AccountStatus:  db.NotReady,
				LastModifiedOn: timeNow,
			})
			require.Nil(t, err)

			// Create an Assignment Entry
			_, err = dbSvc.PutAccountAssignment(db.RedboxAccountAssignment{
				UserID:           userID,
				AccountID:        acctID,
				AssignmentStatus: db.Decommissioned,
				CreatedOn:        timeNow,
				LastModifiedOn:   timeNow,
			})
			require.Nil(t, err)

			// Create the Provision Request Body
			body := leaseRequest{
				UserID:    userID,
				AccountID: acctID,
			}

			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + "/leases",
				json:   body,
			})

			// Verify response code
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// Parse response json
			data := parseResponseJSON(t, resp)

			// Verify error response json
			// Get nested json in response json
			errResp := data["error"].(map[string]interface{})
			require.Equal(t, "ClientError", errResp["code"].(string))
			require.Equal(t, "Account Assignment is not active for user - 123",
				errResp["message"].(string))
		})

	})

	t.Run("Get Accounts", func(t *testing.T) {

		t.Run("returns a list of accounts", func(t *testing.T) {
			defer truncateAccountTable(t, dbSvc)

			expectedID := "234523456"
			account := *newAccount(expectedID, 1561382513)
			err := dbSvc.PutAccount(account)
			require.Nil(t, err)

			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/accounts",
			})

			accounts := parseResponseArrayJSON(t, resp)
			require.True(t, true, len(*accounts) > 0)
			require.Equal(t, (*accounts)[0].ID, expectedID, "The ID of the returns record should match the expected ID")
		})

	})

	t.Run("Get Account By ID", func(t *testing.T) {

		t.Run("returns an account by ID", func(t *testing.T) {
			defer truncateAccountTable(t, dbSvc)

			expectedID := "234523456"
			account := *newAccount(expectedID, 1561382513)
			err := dbSvc.PutAccount(account)
			require.Nil(t, err)

			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/accounts/234523456",
			})

			parseAccount := parseResponseJSON(t, resp)
			require.True(t, true, len(parseAccount) > 0)
			require.Equal(t, 200, resp.StatusCode)
			require.Equal(t, expectedID, parseAccount["id"], "The ID of the returned record should match the expected ID")
		})

	})

	t.Run("Create Account", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping tests in short mode. IAM role takes a while to propagate...")
		}

		// Lookup the current AWS Account ID
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
		roleName := "redbox-api-test-admin-role-" + fmt.Sprintf("%v", time.Now().Unix())
		roleRes, err := iamSvc.CreateRole(&iam.CreateRoleInput{
			AssumeRolePolicyDocument: aws.String(assumeRolePolicy),
			Path:                     aws.String("/"),
			RoleName:                 aws.String(roleName),
		})
		require.Nil(t, err)
		adminRoleArn := *roleRes.Role.Arn

		// IAM Role takes a while to propagate....
		time.Sleep(10 * time.Second)

		// Cleanup: Delete the admin role
		defer func() {
			_, err = iamSvc.DeleteRole(&iam.DeleteRoleInput{
				RoleName: aws.String(roleName),
			})
			require.Nil(t, err)
		}()

		t.Run("Creates an account", func(t *testing.T) {
			// Cleanup the accounts table when we're done
			defer truncateAccountTable(t, dbSvc)

			// Add the current account to the account pool
			apiRes := apiRequest(t, &apiRequestInput{
				method: "POST",
				url:    apiURL + "/accounts",
				json: createAccountRequest{
					ID:           currentAccountID,
					AdminRoleArn: adminRoleArn,
				},
			})

			// Check the response
			require.Equal(t, apiRes.StatusCode, 201)
			resJSON := parseResponseJSON(t, apiRes)
			require.Equal(t, currentAccountID, resJSON["id"])
			require.Equal(t, "NotReady", resJSON["accountStatus"])
			require.Equal(t, adminRoleArn, resJSON["adminRoleArn"])
			require.True(t, resJSON["lastModifiedOn"].(float64) > 1561518000)
			require.True(t, resJSON["createdOn"].(float64) > 1561518000)

			// Check that the account is added to the DB
			dbAccount, err := dbSvc.GetAccount(currentAccountID)
			require.Nil(t, err)
			require.Equal(t, dbAccount, &db.RedboxAccount{
				ID:             currentAccountID,
				AccountStatus:  "NotReady",
				LastModifiedOn: int64(resJSON["lastModifiedOn"].(float64)),
				CreatedOn:      int64(resJSON["createdOn"].(float64)),
				AdminRoleArn:   adminRoleArn,
			})

			// Check that we can retrieve the account via the API
			apiRes = apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/accounts/" + currentAccountID,
			})
			require.Equal(t, apiRes.StatusCode, 200)
			resJSON = parseResponseJSON(t, apiRes)
			require.Equal(t, currentAccountID, resJSON["id"])
			require.Equal(t, "NotReady", resJSON["accountStatus"])
			require.True(t, resJSON["lastModifiedOn"].(float64) > 1561518000)
			require.True(t, resJSON["createdOn"].(float64) > 1561518000)
			require.Equal(t, adminRoleArn, resJSON["adminRoleArn"])
		})

	})

	t.Run("Delete Account", func(t *testing.T) {
		accountID := "1234523456"

		t.Run("when the account exists", func(t *testing.T) {
			t.Run("when the account is not assigned", func(t *testing.T) {
				defer truncateAccountTable(t, dbSvc)
				account := *newAccount(accountID, 1561382513)
				err := dbSvc.PutAccount(account)
				require.Nil(t, err)

				resp := apiRequest(t, &apiRequestInput{
					method: "DELETE",
					url:    apiURL + "/accounts/1234523456",
				})

				require.Equal(t, http.StatusNoContent, resp.StatusCode, "it returns a 204")

				foundAccount, err := dbSvc.GetAccount(accountID)
				require.Nil(t, err)
				require.Nil(t, foundAccount, "the account no longer exists")
			})

			t.Run("when the account is assigned", func(t *testing.T) {
				defer truncateAccountTable(t, dbSvc)
				account := db.RedboxAccount{
					ID:             accountID,
					AccountStatus:  db.Assigned,
					LastModifiedOn: 1561382309,
				}
				err := dbSvc.PutAccount(account)
				require.Nil(t, err)

				resp := apiRequest(t, &apiRequestInput{
					method: "DELETE",
					url:    apiURL + "/accounts/1234523456",
				})

				require.Equal(t, http.StatusConflict, resp.StatusCode, "it returns a 409")

				foundAccount, err := dbSvc.GetAccount(accountID)
				require.Nil(t, err)
				require.NotNil(t, foundAccount, "the account still exists")
			})
		})

		t.Run("when the account does not exists", func(t *testing.T) {
			resp := apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + "/accounts/1234523456",
			})

			require.Equal(t, http.StatusNotFound, resp.StatusCode, "it returns a 404")
		})
	})
}

type leaseRequest struct {
	UserID    string `json:"userId"`
	AccountID string `json:"accountId"`
}

type createAccountRequest struct {
	ID           string `json:"id"`
	AdminRoleArn string `json:"adminRoleArn"`
}

type apiRequestInput struct {
	method string
	url    string
	creds  *credentials.Credentials
	region string
	json   interface{}
}

func apiRequest(t *testing.T, input *apiRequestInput) *http.Response {
	// Set defaults
	if input.creds == nil {
		input.creds = credentials.NewChainCredentials([]credentials.Provider{
			&credentials.EnvProvider{},
			&credentials.SharedCredentialsProvider{Filename: "", Profile: ""},
		})
	}
	if input.region == "" {
		input.region = "us-east-1"
	}

	// Create API request
	req, err := http.NewRequest(input.method, input.url, nil)
	require.Nil(t, err)

	// Sign our API request, using sigv4
	// See https://docs.aws.amazon.com/general/latest/gr/sigv4_signing.html
	signer := sigv4.NewSigner(input.creds)
	now := time.Now().Add(time.Duration(30) * time.Second)
	var signedHeaders http.Header

	// If there's a json provided, add it when signing
	// Body does not matter if added before the signing, it will be overwritten
	if input.json != nil {
		payload, err := json.Marshal(input.json)
		require.Nil(t, err)
		req.Header.Set("Content-Type", "application/json")
		signedHeaders, err = signer.Sign(req, bytes.NewReader(payload),
			"execute-api", input.region, now)
	} else {
		signedHeaders, err = signer.Sign(req, nil, "execute-api",
			input.region, now)
	}
	require.Nil(t, err)
	require.NotNil(t, signedHeaders)

	// Send the API requests
	// resp, err := http.DefaultClient.Do(req)
	httpClient := http.Client{
		Timeout: 60 * time.Second,
	}
	resp, err := httpClient.Do(req)
	require.Nil(t, err)

	return resp
}

func parseResponseJSON(t *testing.T, resp *http.Response) map[string]interface{} {
	defer resp.Body.Close()
	var data map[string]interface{}

	body, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)

	err = json.Unmarshal([]byte(body), &data)
	require.Nil(t, err)

	return data
}

func parseResponseArrayJSON(t *testing.T, resp *http.Response) *[]response.AccountResponse {
	defer resp.Body.Close()
	data := &[]response.AccountResponse{}
	body, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	err = json.Unmarshal([]byte(body), data)
	require.Nil(t, err, fmt.Sprintf("Unmarshalling response failed: %s; %s", body, err))

	return data
}
