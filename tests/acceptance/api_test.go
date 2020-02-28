package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/cognitoidentity"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/google/uuid"

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
	"github.com/Optum/dce/tests/acceptance/testutil"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

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
		tfOut["accounts_table_name"].(string),
		tfOut["leases_table_name"].(string),
		7,
	)
	dbSvc.ConsistentRead = true

	// Configure the usage service
	usageSvc := usage.New(
		dynamodb.New(
			awsSession,
			aws.NewConfig().WithRegion(tfOut["aws_region"].(string)),
		),
		tfOut["usage_table_name"].(string),
		"StartDate",
		"PrincipalId",
	)

	// Create an adminRole for the test account
	adminRoleName := "dce-api-test-admin-role-" + fmt.Sprintf("%v", time.Now().Unix())
	costExplorerPolicyName := "dce-api-test-ce-full-access-" + fmt.Sprintf("%v", time.Now().Unix())
	costExplorerPolicyDocument := `{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Action": "ce:*",
						"Resource": "*"
					}
				]
			}`
	cePolicy := createPolicy(t, awsSession, costExplorerPolicyName, costExplorerPolicyDocument)
	policies := []string{
		"arn:aws:iam::aws:policy/IAMFullAccess",
		*cePolicy.Arn,
	}
	adminRoleRes := createAdminRole(t, awsSession, adminRoleName, policies)
	accountID := adminRoleRes.accountID
	adminRoleArn := adminRoleRes.adminRoleArn

	// Cleanup tables before and after tests
	truncateAccountTable(t, dbSvc)
	truncateLeaseTable(t, dbSvc)
	truncateUsageTable(t, usageSvc)
	defer truncateAccountTable(t, dbSvc)
	defer truncateLeaseTable(t, dbSvc)
	defer truncateUsageTable(t, usageSvc)
	defer deletePolicy(t, *cePolicy.Arn)
	defer deleteAdminRole(t, adminRoleName, policies)

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
			apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases",
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Defaults to returning 200
					assert.Equal(r, http.StatusOK, apiResp.StatusCode)
				},
			})
		})

	})

	t.Run("api_execute_admin policy", func(t *testing.T) {

		t.Run("should allow executing DCE APIs", func(t *testing.T) {
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
			roleName := "dce-api-execute-test-role-" + fmt.Sprintf("%v", time.Now().Unix())
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

			// Assume the roleRes we just created
			roleCreds := NewCredentials(t, awsSession, *roleRes.Role.Arn)

			// Attempt to hit the API with using our assumed role
			apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases",
				creds:  roleCreds,
				// This can take a while to propagate
				maxAttempts: 30,
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Defaults to not being unauthorized
					assert.NotEqual(r, http.StatusForbidden, apiResp.StatusCode,
						"Should not return an IAM authorization error")
				},
			})

		})

	})

	t.Run("API permissions are properly configured for Users", func(t *testing.T) {
		// Don't run this test, if using `go test -short` flag
		if testing.Short() {
			t.Skip("Skipping tests in short mode. IAM role takes a while to propagate...")
		}

		// Grab policy name from Terraform outputs
		policyArn := terraform.Output(t, tfOpts, "role_user_policy")
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
		roleName := "dce-api-execute-test-role-" + fmt.Sprintf("%v", time.Now().Unix())
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

		//time.Sleep(10 * time.Second)

		t.Run("should not fail when getting a lease", func(t *testing.T) {
			// Don't run this test, if using `go test -short` flag

			// Assume the roleRes we just created
			roleCreds := NewCredentials(t, awsSession, *roleRes.Role.Arn)

			// Attempt to hit the API with using our assumed role
			apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases",
				creds:  roleCreds,
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Defaults to not being unauthorized
					assert.NotEqual(r, http.StatusForbidden, apiResp.StatusCode,
						"Should not return an IAM authorization error")
				},
			})
		})

		t.Run("should fail when getting accounts", func(t *testing.T) {

			// Assume the roleRes we just created
			roleCreds := NewCredentials(t, awsSession, *roleRes.Role.Arn)

			// Attempt to hit the API with using our assumed role
			apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/accounts",
				creds:  roleCreds,
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Defaults to not being unauthorized
					assert.Equal(r, http.StatusForbidden, apiResp.StatusCode,
						"Should return an IAM authorization error")
				},
			})
		})

		t.Run("should not fail when getting usage", func(t *testing.T) {
			// Don't run this test, if using `go test -short` flag

			// Assume the roleRes we just created
			roleCreds := NewCredentials(t, awsSession, *roleRes.Role.Arn)

			// Attempt to hit the API with using our assumed role
			apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/usage",
				creds:  roleCreds,
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Defaults to not being unauthorized
					assert.NotEqual(r, http.StatusForbidden, apiResp.StatusCode,
						"Should not return an IAM authorization error")
				},
			})
		})

	})

	t.Run("Lease Creation and Deletion", func(t *testing.T) {
		defer truncateAccountTable(t, dbSvc)

		// Add the current account to the account pool
		apiRequest(t, &apiRequestInput{
			method: "POST",
			url:    apiURL + "/accounts",
			json: createAccountRequest{
				ID:           accountID,
				AdminRoleArn: adminRoleArn,
			},
			maxAttempts: 15,
			f: func(r *testutil.R, apiResp *apiResponse) {
				assert.Equal(r, 201, apiResp.StatusCode)
			},
		})

		// Wait for the account to be reset, so we can lease it
		waitForAccountStatus(t, apiURL, accountID, "Ready")

		t.Run("Should not be able to create lease with empty json", func(t *testing.T) {
			// Send an API request
			apiRequest(t, &apiRequestInput{
				method: "POST",
				url:    apiURL + "/leases",
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Verify response code
					assert.Equal(r, http.StatusBadRequest, apiResp.StatusCode)

					// Parse response json
					data := parseResponseJSON(t, apiResp)

					// Verify error response json
					// Get nested json in response json
					err := data["error"].(map[string]interface{})
					assert.Equal(r, "ClientError", err["code"].(string))
					assert.Equal(r, "invalid request parameters",
						err["message"].(string))
				},
			})

		})

		t.Run("Should not be able to destroy lease with empty json", func(t *testing.T) {
			// Send an API request
			apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + "/leases",
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Verify response code
					assert.Equal(r, http.StatusBadRequest, apiResp.StatusCode)

					// Parse response json
					data := parseResponseJSON(t, apiResp)

					// Verify error response json
					// Get nested json in response json
					err := data["error"].(map[string]interface{})
					assert.Equal(r, "ClientError", err["code"].(string))
					assert.Equal(r, "invalid request parameters",
						err["message"].(string))
				},
			})

		})

		t.Run("Should not be able to destroy lease with no leases", func(t *testing.T) {
			// Create the Provision Request Body
			principalID := "user"
			acctID := "123"
			body := leaseRequest{
				PrincipalID: principalID,
				AccountID:   acctID,
			}

			// Send an API request
			apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + "/leases",
				json:   body,
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Verify response code
					assert.Equal(r, http.StatusNotFound, apiResp.StatusCode)

					// Parse response json
					data := parseResponseJSON(t, apiResp)

					// Verify error response json
					// Get nested json in response json
					err := data["error"].(map[string]interface{})
					assert.Equal(r, "NotFoundError", err["code"].(string))
					assert.Equal(r, fmt.Sprintf("lease \"with Principal ID %s and Account ID %s\" not found", principalID, acctID),
						err["message"].(string))
				},
			})

		})

		t.Run("Should not be able to destroy lease with wrong account", func(t *testing.T) {
			// Create an Account Entry
			principalID := "user"
			timeNow := time.Now().Unix()

			// Create an Lease Entry
			_, err = dbSvc.PutLease(db.Lease{
				ID:                    uuid.New().String(),
				PrincipalID:           principalID,
				AccountID:             accountID,
				LeaseStatus:           db.Active,
				CreatedOn:             timeNow,
				LastModifiedOn:        timeNow,
				LeaseStatusModifiedOn: timeNow,
			})
			require.Nil(t, err)

			// Create the Provision Request Body
			wrongAcctID := "456"
			body := leaseRequest{
				PrincipalID: principalID,
				AccountID:   wrongAcctID,
			}

			// Send an API request
			apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + "/leases",
				json:   body,
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Verify response code
					assert.Equal(r, http.StatusNotFound, apiResp.StatusCode)

					// Parse response json
					data := parseResponseJSON(t, apiResp)

					// Verify error response json
					// Get nested json in response json
					errResp := data["error"].(map[string]interface{})
					assert.Equal(r, "NotFoundError", errResp["code"].(string))
					assert.Equal(r, fmt.Sprintf("lease \"with Principal ID %s and Account ID %s\" not found", principalID, wrongAcctID),
						errResp["message"].(string))
				},
			})

		})

		t.Run("Should not be able to destroy lease with NotReady account", func(t *testing.T) {
			// Create an Account Entry
			principalID := "user"
			timeNow := time.Now().Unix()

			// Create an Lease Entry
			_, err = dbSvc.PutLease(db.Lease{
				ID:                    uuid.New().String(),
				PrincipalID:           principalID,
				AccountID:             accountID,
				LeaseStatus:           db.Inactive,
				CreatedOn:             timeNow,
				LastModifiedOn:        timeNow,
				LeaseStatusModifiedOn: timeNow,
			})
			require.Nil(t, err)

			// Create the Provision Request Body
			body := leaseRequest{
				PrincipalID: principalID,
				AccountID:   accountID,
			}

			// Send an API request
			apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + "/leases",
				json:   body,
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Verify response code
					assert.Equal(r, http.StatusConflict, apiResp.StatusCode)

					// Parse response json
					data := parseResponseJSON(t, apiResp)

					// Verify error response json
					// Get nested json in response json
					errResp := data["error"].(map[string]interface{})
					assert.Equal(r, "ConflictError", errResp["code"].(string))
					assert.Regexp(t, "leaseStatus: must be active lease", errResp["message"].(string))
				},
			})

		})

		t.Run("Cognito user should not be able to create, get, list, or delete a lease for another user", func(t *testing.T) {
			///////////
			// Setup //
			///////////
			// Create cognito users
			cognitoUser1 := NewCognitoUser(t, tfOut, awsSession, accountID)
			defer cognitoUser1.delete(t, tfOut, awsSession)
			cognitoUser2 := NewCognitoUser(t, tfOut, awsSession, accountID)
			defer cognitoUser2.delete(t, tfOut, awsSession)

			//////////////////
			// Create Lease //
			//////////////////
			// Cognito User 2 tries to create lease for Cognito User 1 and gets 401
			apiRequest(t, &apiRequestInput{
				method: "POST",
				url:    apiURL + "/leases",
				json: struct {
					PrincipalID              string   `json:"principalId"`
					BudgetAmount             float64  `json:"budgetAmount"`
					BudgetCurrency           string   `json:"budgetCurrency"`
					BudgetNotificationEmails []string `json:"budgetNotificationEmails"`
				}{
					PrincipalID:              cognitoUser1.Username,
					BudgetAmount:             100,
					BudgetCurrency:           "USD",
					BudgetNotificationEmails: []string{"test@optum.com"},
				},
				f: func(r *testutil.R, apiResp *apiResponse) {
					assert.Equalf(r, http.StatusUnauthorized, apiResp.StatusCode, "%v", apiResp.json)
				},
				maxAttempts: 3,
				creds:       credentials.NewStaticCredentialsFromCreds(cognitoUser2.UserCredsValue),
			})
			// Cognito User 1 creates a lease for themself
			resp := apiRequest(t, &apiRequestInput{
				method: "POST",
				url:    apiURL + "/leases",
				json: struct {
					PrincipalID              string   `json:"principalId"`
					BudgetAmount             float64  `json:"budgetAmount"`
					BudgetCurrency           string   `json:"budgetCurrency"`
					BudgetNotificationEmails []string `json:"budgetNotificationEmails"`
				}{
					PrincipalID:              cognitoUser1.Username,
					BudgetAmount:             100.29,
					BudgetCurrency:           "USD",
					BudgetNotificationEmails: []string{"test@optum.com"},
				},
				f: func(r *testutil.R, apiResp *apiResponse) {
					assert.Equalf(r, http.StatusCreated, apiResp.StatusCode, "%v", apiResp.json)
				},
				maxAttempts: 3,
				creds:       credentials.NewStaticCredentialsFromCreds(cognitoUser1.UserCredsValue),
			})
			createLeaseOutput := parseResponseJSON(t, resp)
			///////////////
			// Get Lease //
			///////////////
			// Cognito User 2 should get 401
			apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + fmt.Sprintf("/leases/%s", createLeaseOutput["id"]),
				f: func(r *testutil.R, apiResp *apiResponse) {
					assert.Equal(r, http.StatusUnauthorized, apiResp.StatusCode)
				},
				maxAttempts: 3,
				creds:       credentials.NewStaticCredentialsFromCreds(cognitoUser2.UserCredsValue),
			})
			// Cognito User 1 should get 200
			apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + fmt.Sprintf("/leases/%s", createLeaseOutput["id"]),
				f: func(r *testutil.R, apiResp *apiResponse) {
					assert.Equal(r, http.StatusOK, apiResp.StatusCode)
				},
				maxAttempts: 3,
				creds:       credentials.NewStaticCredentialsFromCreds(cognitoUser1.UserCredsValue),
			})

			/////////////////
			// List Leases //
			/////////////////
			// Cognito User 2 should get empty list when listing leases
			apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases",
				f: func(r *testutil.R, apiResp *apiResponse) {

					// Assert
					respList := parseResponseArrayJSON(t, apiResp)
					assert.Equalf(r, 0, len(respList), "%v", apiResp.json)
					assert.Equalf(r, http.StatusOK, apiResp.StatusCode, "%v", apiResp.json)
				},
				maxAttempts: 3,
				creds:       credentials.NewStaticCredentialsFromCreds(cognitoUser2.UserCredsValue),
			})
			// Cognito User 1 should get a list containing their single lease
			apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases",
				f: func(r *testutil.R, apiResp *apiResponse) {

					// Assert
					respList := parseResponseArrayJSON(t, apiResp)
					assert.Equalf(r, 1, len(respList), "%v", apiResp.json)
					assert.Equalf(r, http.StatusOK, apiResp.StatusCode, "%v", apiResp.json)
				},
				maxAttempts: 3,
				creds:       credentials.NewStaticCredentialsFromCreds(cognitoUser1.UserCredsValue),
			})

			//////////////////
			// Delete Lease //
			//////////////////
			// Cognito User 2 should get 401
			// -> Delete by accountID and PrincipalID
			apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + "/leases",
				json: leaseRequest{
					PrincipalID: cognitoUser1.Username,
					AccountID:   accountID,
				},
				f: func(r *testutil.R, apiResp *apiResponse) {
					assert.Equal(r, http.StatusUnauthorized, apiResp.StatusCode)
				},
				maxAttempts: 3,
				creds:       credentials.NewStaticCredentialsFromCreds(cognitoUser2.UserCredsValue),
			})
			// -> Delete by leaseID
			apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + fmt.Sprintf("/leases/%s", createLeaseOutput["id"]),
				f: func(r *testutil.R, apiResp *apiResponse) {
					assert.Equal(r, http.StatusUnauthorized, apiResp.StatusCode)
				},
				maxAttempts: 3,
				creds:       credentials.NewStaticCredentialsFromCreds(cognitoUser2.UserCredsValue),
			})
			// Cognito User 1 should delete their own lease
			apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + fmt.Sprintf("/leases/%s", createLeaseOutput["id"]),
				f: func(r *testutil.R, apiResp *apiResponse) {
					assert.Equal(r, http.StatusOK, apiResp.StatusCode)
				},
				maxAttempts: 3,
				creds:       credentials.NewStaticCredentialsFromCreds(cognitoUser1.UserCredsValue),
			})
		})

		t.Run("Should not be able to create lease with no available accounts", func(t *testing.T) {
			// Create the Provision Request Body
			principalID := "user"
			body := leaseRequest{
				PrincipalID: principalID,
			}

			// Send an API request
			apiRequest(t, &apiRequestInput{
				method: "POST",
				url:    apiURL + "/leases",
				json:   body,
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Verify response code
					assert.Equal(r, http.StatusInternalServerError, apiResp.StatusCode)

					// Parse response json
					data := parseResponseJSON(t, apiResp)

					// Verify error response json
					// Get nested json in response json
					err := data["error"].(map[string]interface{})
					assert.Equal(r, "ServerError", err["code"].(string))
					assert.Equal(r, "No Available accounts at this moment",
						err["message"].(string))
				},
			})

		})

		t.Run("Should be able to create and destroy and lease by ID", func(t *testing.T) {
			defer truncateLeaseTable(t, dbSvc)

			// Wait for the account to be reset, so we can lease it
			waitForAccountStatus(t, apiURL, accountID, "Ready")

			expiresOn := time.Now().AddDate(0, 0, 6).Unix()

			body := inputLeaseRequest{
				PrincipalID:              "user1",
				BudgetAmount:             200.00,
				BudgetCurrency:           "USD",
				BudgetNotificationEmails: []string{"test1@test.com"},
				ExpiresOn:                expiresOn,
			}

			// Create a lease
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
			require.Equal(t, "user1", data["principalId"].(string))
			require.Equal(t, accountID, data["accountId"].(string))
			require.Equal(t, string(db.Active),
				data["leaseStatus"].(string))
			require.NotNil(t, data["createdOn"])
			require.NotNil(t, data["lastModifiedOn"])
			require.NotNil(t, data["leaseStatusModifiedOn"])

			// Delete the lease
			resp = apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + fmt.Sprintf("/leases/%s", data["id"]),
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Verify response code
					assert.Equal(r, http.StatusOK, apiResp.StatusCode)
				},
			})

			// Parse response json
			data = parseResponseJSON(t, resp)

			// Verify provisioned response json
			assert.Equal(t, "user", data["principalId"].(string))
			assert.Equal(t, accountID, data["accountId"].(string))
			assert.Equal(t, string(db.Inactive),
				data["leaseStatus"].(string))
			assert.NotNil(t, data["createdOn"])
			assert.NotNil(t, data["lastModifiedOn"])
			assert.NotNil(t, data["leaseStatusModifiedOn"])

		})

		t.Run("Should be able to create and destroy a lease", func(t *testing.T) {
			defer truncateLeaseTable(t, dbSvc)

			// Wait for the account to be reset, so we can lease it
			waitForAccountStatus(t, apiURL, accountID, "Ready")

			// Create the Provision Request Body
			principalID := "user"
			expiresOn := time.Now().AddDate(0, 0, 6).Unix()

			body := inputLeaseRequest{
				PrincipalID:              principalID,
				BudgetAmount:             200.00,
				BudgetCurrency:           "USD",
				BudgetNotificationEmails: []string{"test1@test.com"},
				ExpiresOn:                expiresOn,
			}

			// Create the lease
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
			require.Equal(t, principalID, data["principalId"].(string))
			require.Equal(t, accountID, data["accountId"].(string))
			require.Equal(t, string(db.Active),
				data["leaseStatus"].(string))
			require.NotNil(t, data["createdOn"])
			require.NotNil(t, data["lastModifiedOn"])
			require.NotNil(t, data["leaseStatusModifiedOn"])

			// Account should be marked as status=Leased
			waitForAccountStatus(t, apiURL, accountID, "Leased")

			// Delete the lease
			resp = apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + "/leases",
				json: leaseRequest{
					PrincipalID: principalID,
					AccountID:   accountID,
				},
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Verify response code
					assert.Equal(r, http.StatusOK, apiResp.StatusCode)
				},
			})

			// Parse response json
			data = parseResponseJSON(t, resp)

			// Verify provisioned response json
			assert.Equal(t, principalID, data["principalId"].(string))
			assert.Equal(t, accountID, data["accountId"].(string))
			assert.Equal(t, string(db.Inactive),
				data["leaseStatus"].(string))
			assert.NotNil(t, data["createdOn"])
			assert.NotNil(t, data["lastModifiedOn"])
			assert.NotNil(t, data["leaseStatusModifiedOn"])

			// Account should be marked as status=NotReady
			waitForAccountStatus(t, apiURL, accountID, "NotReady")
		})

	})

	t.Run("Account Creation Deletion Flow", func(t *testing.T) {
		// Make sure the DB is clean
		truncateDBTables(t, dbSvc)
		truncateUsageTable(t, usageSvc)
		// Cleanup the DB when we're done
		defer truncateDBTables(t, dbSvc)
		defer truncateUsageTable(t, usageSvc)

		t.Run("STEP: Create Account", func(t *testing.T) {

			// Add the current account to the account pool
			createAccountRes := apiRequest(t, &apiRequestInput{
				method: "POST",
				url:    apiURL + "/accounts",
				json: createAccountRequest{
					ID:           accountID,
					AdminRoleArn: adminRoleArn,
				},
				maxAttempts: 15,
				f: func(r *testutil.R, apiResp *apiResponse) {
					assert.Equal(r, 201, apiResp.StatusCode)
				},
			})

			// Check the response
			postResJSON := parseResponseJSON(t, createAccountRes)
			require.Equal(t, accountID, postResJSON["id"])
			require.Equal(t, "NotReady", postResJSON["accountStatus"])
			require.Equal(t, adminRoleArn, postResJSON["adminRoleArn"])
			expectedPrincipalRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, tfOut["principal_role_name"])
			require.Equal(t, expectedPrincipalRoleArn, postResJSON["principalRoleArn"])
			require.True(t, postResJSON["lastModifiedOn"].(float64) > 1561518000)
			require.True(t, postResJSON["createdOn"].(float64) > 1561518000)

			// Check that the account is added to the DB
			dbAccount, err := dbSvc.GetAccount(accountID)
			require.Nil(t, err)
			require.Equal(t, &db.Account{
				ID:                  accountID,
				AccountStatus:       "NotReady",
				LastModifiedOn:      int64(postResJSON["lastModifiedOn"].(float64)),
				CreatedOn:           int64(postResJSON["createdOn"].(float64)),
				AdminRoleArn:        adminRoleArn,
				PrincipalRoleArn:    expectedPrincipalRoleArn,
				PrincipalPolicyHash: dbAccount.PrincipalPolicyHash,
			}, dbAccount)

			// Check that the IAM Principal Role was created
			// Lookup the principal IAM Role
			iamSvc := iam.New(awsSession)
			roleArn, err := arn.Parse(postResJSON["principalRoleArn"].(string))
			require.Nil(t, err)
			roleName := strings.Split(roleArn.Resource, "/")[1]
			_, err = iamSvc.GetRole(&iam.GetRoleInput{
				RoleName: aws.String(roleName),
			})
			require.Nil(t, err)

			// Check the Role policies
			res, err := iamSvc.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
				RoleName: aws.String(roleName),
			})
			require.Nil(t, err)
			require.Len(t, res.AttachedPolicies, 1)
			principalPolicyArn := res.AttachedPolicies[0].PolicyArn

			t.Run("STEP: Get Account by ID", func(t *testing.T) {
				// Send GET /accounts/id
				apiRequest(t, &apiRequestInput{
					method: "GET",
					url:    apiURL + "/accounts/" + accountID,
					f: func(r *testutil.R, apiResp *apiResponse) {
						// Check the GET /accounts response
						assert.Equal(r, apiResp.StatusCode, 200)
						getResJSON := apiResp.json.(map[string]interface{})
						assert.Equal(r, accountID, getResJSON["id"])
						assert.Equal(r, "NotReady", getResJSON["accountStatus"])
						assert.Equal(r, adminRoleArn, getResJSON["adminRoleArn"])
						expectedPrincipalRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, tfOut["principal_role_name"])
						assert.Equal(r, expectedPrincipalRoleArn, getResJSON["principalRoleArn"])
						assert.True(r, getResJSON["lastModifiedOn"].(float64) > 1561518000)
						assert.True(r, getResJSON["createdOn"].(float64) > 1561518000)
					},
				})

			})

			t.Run("STEP: List Accounts", func(t *testing.T) {
				// Send GET /accounts
				apiRequest(t, &apiRequestInput{
					method: "GET",
					url:    apiURL + "/accounts",
					f: func(r *testutil.R, apiResp *apiResponse) {
						// Check the response
						assert.Equal(r, apiResp.StatusCode, 200)
						listResJSON := parseResponseArrayJSON(t, apiResp)
						accountJSON := listResJSON[0]
						assert.Equal(r, accountID, accountJSON["id"])
						assert.Equal(r, "NotReady", accountJSON["accountStatus"])
						assert.Equal(r, adminRoleArn, accountJSON["adminRoleArn"])
						expectedPrincipalRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, tfOut["principal_role_name"])
						assert.Equal(r, expectedPrincipalRoleArn, accountJSON["principalRoleArn"])
						assert.True(r, accountJSON["lastModifiedOn"].(float64) > 1561518000)
						assert.True(r, accountJSON["createdOn"].(float64) > 1561518000)
					},
				})

			})

			t.Run("STEP: Create Lease", func(t *testing.T) {
				// Wait for the account to be reset, so we can lease it
				waitForAccountStatus(t, apiURL, accountID, "Ready")

				var budgetAmount float64 = 300
				var budgetNotificationEmails = []string{"test@test.com"}

				// Create a lease
				res := apiRequest(t, &apiRequestInput{
					method: "POST",
					url:    apiURL + "/leases",
					json: struct {
						PrincipalID              string   `json:"principalId"`
						BudgetAmount             float64  `json:"budgetAmount"`
						BudgetCurrency           string   `json:"budgetCurrency"`
						BudgetNotificationEmails []string `json:"budgetNotificationEmails"`
					}{
						PrincipalID:              "test-user",
						BudgetAmount:             budgetAmount,
						BudgetCurrency:           "USD",
						BudgetNotificationEmails: budgetNotificationEmails,
					},
					f: func(r *testutil.R, apiResp *apiResponse) {
						assert.Equalf(r, 201, apiResp.StatusCode, "%v", apiResp.json)
					},
				})

				resJSON := parseResponseJSON(t, res)

				s := make([]interface{}, len(budgetNotificationEmails))
				for i, v := range budgetNotificationEmails {
					s[i] = v
				}

				require.Contains(t, resJSON, "id")
				require.Equal(t, "test-user", resJSON["principalId"])
				require.Equal(t, accountID, resJSON["accountId"])
				require.Equal(t, "Active", resJSON["leaseStatus"])
				require.NotNil(t, resJSON["createdOn"])
				require.NotNil(t, resJSON["lastModifiedOn"])
				require.Equal(t, budgetAmount, resJSON["budgetAmount"])
				require.Equal(t, "USD", resJSON["budgetCurrency"])
				require.Equal(t, s, resJSON["budgetNotificationEmails"])
				_, err = uuid.Parse(fmt.Sprintf("%v", resJSON["id"]))
				require.Nil(t, err)
				require.NotNil(t, resJSON["leaseStatusModifiedOn"])

				// Check the lease is created
				res = apiRequest(t, &apiRequestInput{
					method: "GET",
					url:    apiURL + "/leases/" + resJSON["id"].(string),
					f: func(r *testutil.R, apiResp *apiResponse) {
						assert.Equal(r, 200, apiResp.StatusCode)
					},
				})
				leaseJSON := parseResponseJSON(t, res)
				require.Equal(t, accountID, leaseJSON["accountId"])

				// Account should be marked as status=Leased
				apiRequest(t, &apiRequestInput{
					method: "GET",
					url:    apiURL + "/accounts/" + accountID,
					f: func(r *testutil.R, apiResp *apiResponse) {
						status := responseJSONString(r, apiResp, "accountStatus")
						assert.Equal(r, "Leased", status)
					},
				})

				t.Run("STEP: Create duplicate lease for same principal (should fail)", func(t *testing.T) {
					// Create a lease
					res = apiRequest(t, &apiRequestInput{
						method:      "POST",
						url:         apiURL + "/leases",
						maxAttempts: 1,
						json: map[string]interface{}{
							"principalId":               "test-user",
							"budgetAmount":              800,
							"budgetCurrency":            "USD",
							"budgetNotificationsEmails": []string{"test@example.com"},
						},
					})
					require.Equal(t, 409, res.StatusCode)
					require.Equal(t, map[string]interface{}{
						"error": map[string]interface{}{
							"code":    "ClientError",
							"message": fmt.Sprintf("Principal already has an active lease for account %s", accountID),
						},
					}, parseResponseJSON(t, res))

					// Make sure there's still only one lease in the system
					res = apiRequest(t, &apiRequestInput{
						method: "GET",
						url:    apiURL + "/leases",
						f: func(r *testutil.R, apiResp *apiResponse) {
							assert.Equal(r, 200, apiResp.StatusCode)
						},
					})
					leasesData := parseResponseArrayJSON(t, res)
					require.Len(t, leasesData, 1)
				})

				t.Run("STEP: Delete Account (with Lease)", func(t *testing.T) {
					// Request a lease
					apiRequest(t, &apiRequestInput{
						method: "DELETE",
						url:    apiURL + "/accounts/" + accountID,
						json: struct {
							PrincipalID string `json:"principalId"`
						}{
							PrincipalID: "test-user",
						},
						f: func(r *testutil.R, apiResp *apiResponse) {
							assert.Equal(r, 409, apiResp.StatusCode)
						},
					})

				})

				t.Run("STEP: Delete Lease", func(t *testing.T) {
					// Delete the lease
					apiRequest(t, &apiRequestInput{
						method: "DELETE",
						url:    apiURL + "/leases",
						json: struct {
							PrincipalID string `json:"principalId"`
							AccountID   string `json:"accountId"`
						}{
							PrincipalID: "test-user",
							AccountID:   accountID,
						},
						f: func(r *testutil.R, apiResp *apiResponse) {
							assert.Equal(r, 200, apiResp.StatusCode)
						},
					})

					// Check the lease is decommissioned
					resp := apiRequest(t, &apiRequestInput{
						method: "GET",
						url:    apiURL + fmt.Sprintf("/leases?principalId=test-user&accountId=%s", accountID),
						json:   nil,
					})

					results := parseResponseArrayJSON(t, resp)
					assert.Equal(t, 200, resp.StatusCode)
					assert.Equal(t, 1, len(results), "one lease should be returned")
					assert.Equal(t, "Inactive", results[0]["leaseStatus"])

					// Account status should change from Leased --> NotReady
					waitForAccountStatus(t, apiURL, accountID, "NotReady")

					t.Run("STEP: Delete Account", func(t *testing.T) {
						// Delete the account
						apiRequest(t, &apiRequestInput{
							method: "DELETE",
							url:    apiURL + "/accounts/" + accountID,
							f: func(r *testutil.R, apiResp *apiResponse) {
								assert.Equal(r, 204, apiResp.StatusCode)
							},
						})

						// Attempt to get the deleted account (should 404)
						apiRequest(t, &apiRequestInput{
							method: "GET",
							url:    apiURL + "/accounts/" + accountID,
							f: func(r *testutil.R, apiResp *apiResponse) {
								assert.Equal(t, 404, apiResp.StatusCode)
							},
						})

						// Check that the Principal Role was deleted
						_, err = iamSvc.GetRole(&iam.GetRoleInput{
							RoleName: aws.String(roleName),
						})
						require.NotNil(t, err)
						require.Equal(t, iam.ErrCodeNoSuchEntityException, err.(awserr.Error).Code())

						// Check that the Principal Policy was deleted
						_, err = iamSvc.GetPolicy(&iam.GetPolicyInput{
							PolicyArn: principalPolicyArn,
						})
						require.NotNil(t, err)
						require.Equal(t, iam.ErrCodeNoSuchEntityException, err.(awserr.Error).Code())
					})
				})

			})

		})

	})

	t.Run("Create account with metadata", func(t *testing.T) {
		// Make sure the DB is clean
		truncateDBTables(t, dbSvc)
		defer truncateDBTables(t, dbSvc)

		// Create an account with metadata
		res := apiRequest(t, &apiRequestInput{
			method: "POST",
			url:    apiURL + "/accounts",
			json: map[string]interface{}{
				"id":           accountID,
				"adminRoleArn": adminRoleArn,
				"metadata": map[string]interface{}{
					"foo": map[string]interface{}{
						"bar": "baz",
					},
					"hello": "you",
				},
			},
			f: func(r *testutil.R, apiResp *apiResponse) {
				assert.Equal(r, 201, apiResp.StatusCode)
			},
		})

		// Check the response
		require.Equal(t, res.StatusCode, 201)
		resJSON := parseResponseJSON(t, res)
		require.Equal(t, map[string]interface{}{
			"foo": map[string]interface{}{
				"bar": "baz",
			},
			"hello": "you",
		}, resJSON["metadata"])

		// Check the DB record
		dbAccount, err := dbSvc.GetAccount(accountID)
		require.Nil(t, err)
		require.Equal(t, map[string]interface{}{
			"foo": map[string]interface{}{
				"bar": "baz",
			},
			"hello": "you",
		}, dbAccount.Metadata)

		// Check the GET /accounts API response
		getRes := apiRequest(t, &apiRequestInput{
			method: "GET",
			url:    apiURL + "/accounts/" + accountID,
			f: func(r *testutil.R, apiResp *apiResponse) {
				assert.Equal(r, 200, apiResp.StatusCode)
			},
		})
		require.Equal(t, getRes.StatusCode, 200)
		getResJSON := parseResponseJSON(t, getRes)
		require.Equal(t, map[string]interface{}{
			"foo": map[string]interface{}{
				"bar": "baz",
			},
			"hello": "you",
		}, getResJSON["metadata"])
	})

	t.Run("Create multiple leases on one account", func(t *testing.T) {

		// Create an account
		apiRequest(t, &apiRequestInput{
			method: "POST",
			url:    apiURL + "/accounts",
			json: map[string]interface{}{
				"id":           accountID,
				"adminRoleArn": adminRoleArn,
			},
			maxAttempts: 1,
			f: func(r *testutil.R, apiResp *apiResponse) {
				assert.Equal(r, 201, apiResp.StatusCode)
			},
		})

		// Wait for the account to be ready
		log.Printf("Account created. Waiting for initial reset to complete")
		waitForAccountStatus(t, apiURL, accountID, "Ready")

		// Make 3 leases in a row
		for i := range [3]int{} {
			log.Printf("Lease attempt %d", i)

			// Create a lease
			res := apiRequest(t, &apiRequestInput{
				method: "POST",
				url:    apiURL + "/leases",
				json: map[string]interface{}{
					"principalId":    "test-user",
					"budgetAmount":   500,
					"budgetCurrency": "EUR",
					"expiresOn":      time.Now().Unix() + 1000,
				},
				maxAttempts: 1,
				f: func(r *testutil.R, apiResp *apiResponse) {
					assert.Equal(r, 201, apiResp.StatusCode)
				},
			})
			require.Equal(t, 201, res.StatusCode)

			// Account should be Leased
			log.Println("Lease created. Waiting for account to be marked 'Leased'")
			waitForAccountStatus(t, apiURL, accountID, "Leased")

			// Destroy the lease
			res = apiRequest(t, &apiRequestInput{
				method:      "DELETE",
				url:         apiURL + "/leases",
				maxAttempts: 1,
				json: map[string]interface{}{
					"principalId": "test-user",
					"accountId":   accountID,
				},
				f: func(r *testutil.R, apiResp *apiResponse) {
					assert.Equal(r, 200, apiResp.StatusCode, apiResp.json)
				},
			})
			require.Equal(t, 200, res.StatusCode)

			// Account should be NotReady, while nuke runs
			log.Println("Lease ended. Waiting for account to be marked 'NotReady'")
			waitForAccountStatus(t, apiURL, accountID, "NotReady")

			// Account should go back to Ready, after nuke is complete
			log.Println("Lease ended. Waiting for nuke to complete")
			waitForAccountStatus(t, apiURL, accountID, "Ready")
		}

	})

	t.Run("Delete Account", func(t *testing.T) {

		t.Run("when the account does not exists", func(t *testing.T) {
			apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + "/accounts/1234523456",
				f: func(r *testutil.R, apiResp *apiResponse) {
					assert.Equal(r, http.StatusNotFound, apiResp.StatusCode, "it returns a 404")
				},
			})
		})

	})

	t.Run("Update Account", func(t *testing.T) {
		// Make sure the DB is clean
		truncateDBTables(t, dbSvc)
		defer truncateDBTables(t, dbSvc)

		// Create an account
		_ = apiRequest(t, &apiRequestInput{
			method: "POST",
			url:    apiURL + "/accounts",
			json: map[string]interface{}{
				"id":           accountID,
				"adminRoleArn": adminRoleArn,
			},
			f: statusCodeAssertion(201),
		})

		t.Run("should update an account's metadata", func(t *testing.T) {
			// wait a second, so we can check that timestamps are updated
			time.Sleep(time.Second)

			// PUT /accounts/:id
			// with update to metadata
			res := apiRequest(t, &apiRequestInput{
				method: "PUT",
				url:    apiURL + "/accounts/" + accountID,
				json: map[string]interface{}{
					"metadata": map[string]interface{}{
						"foo": "bar",
					},
				},
				f: statusCodeAssertion(200),
			})

			// Check the JSON response
			resJSON := parseResponseJSON(t, res)
			require.Equal(t, map[string]interface{}{
				"foo": "bar",
			}, resJSON["metadata"], "Response includes updated metadata")
			require.True(t, resJSON["lastModifiedOn"].(float64) > resJSON["createdOn"].(float64),
				"should update lastModifiedOn timestamp")

			// Check the DB record, to make sure it's updated
			account, err := dbSvc.GetAccount(accountID)
			require.Nil(t, err)

			require.Equal(t, map[string]interface{}{
				"foo": "bar",
			}, account.Metadata, "db record metadata is updated")
			require.True(t, account.LastModifiedOn > account.CreatedOn,
				"should update lastModifiedOn timestamp")
		})

		t.Run("should fail if the new adminRoleArn is not assumable", func(t *testing.T) {
			// PUT /accounts/:id
			// with invalid adminRoleArn
			res := apiRequest(t, &apiRequestInput{
				method: "PUT",
				url:    apiURL + "/accounts/" + accountID,
				json: map[string]interface{}{
					"adminRoleArn": adminRoleArn + "not-valid-role",
				},
				f: statusCodeAssertion(400),
			})

			resJSON := parseResponseJSON(t, res)
			require.Equal(t, map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "RequestValidationError",
					"message": "account validation error: adminRoleArn: must be an admin role arn that can be assumed.",
				},
			}, resJSON)
		})

		t.Run("should return a 404 if the account doesn't exist", func(t *testing.T) {
			// PUT /accounts/:id
			// with invalid adminRoleArn
			res := apiRequest(t, &apiRequestInput{
				method: "PUT",
				url:    apiURL + "/accounts/123456789012",
				json: map[string]interface{}{
					"metadata": map[string]interface{}{
						"foo": "bar",
					},
				},
			})
			require.Equal(t, 404, res.StatusCode)

			resJSON := parseResponseJSON(t, res)
			require.Equal(t, map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "NotFoundError",
					"message": "account \"123456789012\" not found",
				},
			}, resJSON)
		})

	})

	t.Run("Get Usage api", func(t *testing.T) {

		t.Run("Should get an error for invalid date format", func(t *testing.T) {

			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/usage?startDate=2019-09-2&endDate=2019-09-2",
				json:   nil,
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Verify response code
					assert.Equal(r, http.StatusBadRequest, apiResp.StatusCode)
				},
			})

			// Parse response json
			data := parseResponseJSON(t, resp)

			// Verify error response json
			// Get nested json in response json
			errResp := data["error"].(map[string]interface{})
			require.Equal(t, "RequestValidationError", errResp["code"].(string))
			require.Equal(t, "Failed to parse usage start date: strconv.ParseInt: parsing \"2019-09-2\": invalid syntax",
				errResp["message"].(string))
		})

		t.Run("Should get an empty json for usage not found for given input date range", func(t *testing.T) {

			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/usage?startDate=1568937600&endDate=1569023999",
				json:   nil,
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Verify response code
					assert.Equal(r, http.StatusOK, apiResp.StatusCode)
				},
			})

			// Parse response json
			data := parseResponseArrayJSON(t, resp)

			// Verify response json
			require.Equal(t, []map[string]interface{}([]map[string]interface{}{}), data)
		})

		t.Run("Should be able to get usage", func(t *testing.T) {

			defer truncateUsageTable(t, usageSvc)
			createUsage(t, apiURL, usageSvc)

			currentDate := time.Now()
			testStartDate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, time.UTC)
			testEndDate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 23, 59, 59, 59, time.UTC)
			var testPrincipalID = "TestUser1"
			var testAccount = "123456789012"

			t.Run("Should be able to get usage by start date and end date", func(t *testing.T) {
				queryString := fmt.Sprintf("/usage?startDate=%d&endDate=%d", testStartDate.Unix(), testEndDate.Unix())
				requestURL := apiURL + queryString

				testutil.Retry(t, 10, 10*time.Millisecond, func(r *testutil.R) {

					resp := apiRequest(t, &apiRequestInput{
						method: "GET",
						url:    requestURL,
						json:   nil,
					})

					// Verify response code
					assert.Equal(r, http.StatusOK, resp.StatusCode)

					// Parse response json
					data := parseResponseArrayJSON(t, resp)

					//Verify response json
					if data[0] != nil {
						usageJSON := data[0]
						assert.Equal(r, "TestUser1", usageJSON["principalId"].(string))
						assert.Equal(r, "123456789012", usageJSON["accountId"].(string))
						assert.Equal(r, 2000.00, usageJSON["costAmount"].(float64))
					}
				})
			})

			t.Run("Should be able to get usage by start date and principalId", func(t *testing.T) {
				queryString := fmt.Sprintf("/usage?startDate=%d&principalId=%s", testStartDate.Unix(), testPrincipalID)
				requestURL := apiURL + queryString

				testutil.Retry(t, 10, 10*time.Millisecond, func(r *testutil.R) {

					resp := apiRequest(t, &apiRequestInput{
						method: "GET",
						url:    requestURL,
						json:   nil,
					})

					// Verify response code
					assert.Equal(r, http.StatusOK, resp.StatusCode)

					// Parse response json
					data := parseResponseArrayJSON(t, resp)

					//Verify response json
					if data[0] != nil {
						usageJSON := data[0]
						assert.Equal(r, "TestUser1", usageJSON["principalId"].(string))
						assert.Equal(r, "123456789012", usageJSON["accountId"].(string))
						assert.Equal(r, 2000.00, usageJSON["costAmount"].(float64))
					}
				})
			})

			t.Run("Should be able to get all usage", func(t *testing.T) {
				queryString := "/usage"
				requestURL := apiURL + queryString

				testutil.Retry(t, 10, 10*time.Millisecond, func(r *testutil.R) {

					resp := apiRequest(t, &apiRequestInput{
						method: "GET",
						url:    requestURL,
						json:   nil,
					})

					// Verify response code
					assert.Equal(r, http.StatusOK, resp.StatusCode)

					// Parse response json
					data := parseResponseArrayJSON(t, resp)

					//Verify response json
					if data[0] != nil {
						usageJSON := data[0]
						assert.Equal(r, "TestUser1", usageJSON["principalId"].(string))
						assert.Equal(r, "123456789012", usageJSON["accountId"].(string))
						assert.Equal(r, 2000.00, usageJSON["costAmount"].(float64))
					}
				})
			})

			t.Run("Get usage when there are no query parameters", func(t *testing.T) {
				resp := apiRequest(t, &apiRequestInput{
					method: "GET",
					url:    apiURL + "/usage",
					json:   nil,
				})

				results := parseResponseArrayJSON(t, resp)
				assert.Equal(t, 5, len(results), "all usage records should be returned")

				// Check one of the result objects, to make sure it looks right
				_, hasAccountID := results[0]["accountId"]
				_, hasPrincipalID := results[0]["principalId"]
				_, hasStartDate := results[0]["startDate"]

				assert.True(t, hasAccountID, "response should be serialized with the accountId property")
				assert.True(t, hasPrincipalID, "response should be serialized with the principalId property")
				assert.True(t, hasStartDate, "response should be serialized with the startDate property")
			})

			t.Run("Get usage when there is an account ID parameter", func(t *testing.T) {
				resp := apiRequest(t, &apiRequestInput{
					method: "GET",
					url:    apiURL + "/usage?accountId=" + testAccount,
					json:   nil,
				})

				results := parseResponseArrayJSON(t, resp)
				assert.Equal(t, 5, len(results), "only five usage records should be returned")
			})

			t.Run("Get usage when there is an principal ID parameter", func(t *testing.T) {
				resp := apiRequest(t, &apiRequestInput{
					method: "GET",
					url:    apiURL + "/usage?principalId=" + testPrincipalID,
					json:   nil,
				})

				results := parseResponseArrayJSON(t, resp)
				assert.Equal(t, 5, len(results), "only five usage records should be returned")
			})

			t.Run("Get usage when there is a limit parameter", func(t *testing.T) {
				resp := apiRequest(t, &apiRequestInput{
					method: "GET",
					url:    apiURL + "/usage?limit=1",
					json:   nil,
				})

				results := parseResponseArrayJSON(t, resp)
				assert.Equal(t, 1, len(results), "only one usage record should be returned")
			})

			t.Run("Get usage when there is a start date parameter", func(t *testing.T) {
				currentDate := time.Now()
				testStartDate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, time.UTC)
				testDate := fmt.Sprint(testStartDate.Unix())
				resp := apiRequest(t, &apiRequestInput{
					method: "GET",
					url:    apiURL + "/usage?startDate=" + testDate,
					json:   nil,
				})

				results := parseResponseArrayJSON(t, resp)
				assert.Equal(t, 1, len(results), "only one usage record should be returned")
			})

			t.Run("Get usage when there is a Link header", func(t *testing.T) {
				nextPageRegex := regexp.MustCompile(`<(.+)>`)

				respOne := apiRequest(t, &apiRequestInput{
					method: "GET",
					url:    apiURL + "/usage?limit=2",
					json:   nil,
				})

				linkHeader, ok := respOne.Header["Link"]
				assert.True(t, ok, "Link header should exist")

				resultsOne := parseResponseArrayJSON(t, respOne)
				assert.Equal(t, 2, len(resultsOne), "only two usage records should be returned")

				nextPage := nextPageRegex.FindStringSubmatch(linkHeader[0])[1]

				_, err := url.ParseRequestURI(nextPage)
				assert.Nil(t, err, "Link header should contain a valid URL")

				respTwo := apiRequest(t, &apiRequestInput{
					method: "GET",
					url:    nextPage,
					json:   nil,
				})

				linkHeader, ok = respTwo.Header["Link"]
				assert.True(t, ok, "Link header should exist")

				resultsTwo := parseResponseArrayJSON(t, respTwo)
				assert.Equal(t, 2, len(resultsTwo), "only two usage records should be returned")

				nextPage = nextPageRegex.FindStringSubmatch(linkHeader[0])[1]

				_, err = url.ParseRequestURI(nextPage)
				assert.Nil(t, err, "Link header should contain a valid URL")

				respThree := apiRequest(t, &apiRequestInput{
					method: "GET",
					url:    nextPage,
					json:   nil,
				})

				_, ok = respThree.Header["Link"]
				assert.False(t, ok, "Link header should not exist in last page")

				resultsThree := parseResponseArrayJSON(t, respThree)
				assert.Equal(t, 1, len(resultsThree), "only one usage record should be returned")

				results := append(resultsOne, resultsTwo...)
				results = append(results, resultsThree...)

				assert.Equal(t, 5, len(results), "All five usage records should be returned")
			})
		})
	})

	t.Run("Get Leases", func(t *testing.T) {

		t.Run("should return empty for no leases", func(t *testing.T) {
			defer truncateLeaseTable(t, dbSvc)

			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases",
				json:   nil,
			})

			results := parseResponseArrayJSON(t, resp)

			assert.Equal(t, results, []map[string]interface{}{}, "API should return []")
		})

		defer truncateLeaseTable(t, dbSvc)

		accountIDOne := "1"
		accountIDTwo := "2"
		principalIDOne := "a"
		principalIDTwo := "b"
		principalIDThree := "c"
		principalIDFour := "d"

		_, err = dbSvc.PutLease(db.Lease{
			ID:                uuid.New().String(),
			AccountID:         accountIDOne,
			PrincipalID:       principalIDOne,
			LeaseStatus:       db.Active,
			LeaseStatusReason: db.LeaseActive,
		})

		assert.Nil(t, err)

		_, err = dbSvc.PutLease(db.Lease{
			ID:                uuid.New().String(),
			AccountID:         accountIDOne,
			PrincipalID:       principalIDTwo,
			LeaseStatus:       db.Active,
			LeaseStatusReason: db.LeaseActive,
		})

		assert.Nil(t, err)

		_, err = dbSvc.PutLease(db.Lease{
			ID:                uuid.New().String(),
			AccountID:         accountIDOne,
			PrincipalID:       principalIDThree,
			LeaseStatus:       db.Inactive,
			LeaseStatusReason: db.LeaseActive,
		})

		assert.Nil(t, err)

		_, err = dbSvc.PutLease(db.Lease{
			ID:                uuid.New().String(),
			AccountID:         accountIDTwo,
			PrincipalID:       principalIDFour,
			LeaseStatus:       db.Active,
			LeaseStatusReason: db.LeaseActive,
		})

		assert.Nil(t, err)

		_, err = dbSvc.PutLease(db.Lease{
			ID:                uuid.New().String(),
			AccountID:         accountIDTwo,
			PrincipalID:       principalIDOne,
			LeaseStatus:       db.Inactive,
			LeaseStatusReason: db.LeaseActive,
		})

		assert.Nil(t, err)

		t.Run("When there are no query parameters", func(t *testing.T) {
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases",
				json:   nil,
			})

			results := parseResponseArrayJSON(t, resp)
			assert.Equal(t, 5, len(results), "all five leases should be returned")

			// Check one of the result objects, to make sure it looks right
			_, hasAccountID := results[0]["accountId"]
			_, hasPrincipalID := results[0]["principalId"]
			_, hasLeaseStatus := results[0]["leaseStatus"]

			assert.True(t, hasAccountID, "response should be serialized with the accountId property")
			assert.True(t, hasPrincipalID, "response should be serialized with the principalId property")
			assert.True(t, hasLeaseStatus, "response should be serialized with the leaseStatus property")
		})

		t.Run("When there is an account ID parameter", func(t *testing.T) {
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases?accountId=" + accountIDOne,
				json:   nil,
			})

			results := parseResponseArrayJSON(t, resp)
			assert.Equal(t, 3, len(results), "only three leases should be returned")
		})

		t.Run("When there is an principal ID parameter", func(t *testing.T) {
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases?principalId=" + principalIDOne,
				json:   nil,
			})

			results := parseResponseArrayJSON(t, resp)
			assert.Equal(t, 2, len(results), "only two leases should be returned")
		})

		t.Run("When there is a principal ID and an Account ID parameter", func(t *testing.T) {
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases?principalId=" + principalIDOne + "&accountId=" + accountIDOne,
				json:   nil,
			})

			results := parseResponseArrayJSON(t, resp)
			assert.Equal(t, 1, len(results), "only one lease should be returned")
		})

		t.Run("When there is no leases found with principal and account don't exist", func(t *testing.T) {
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases?principalId=reallybadprincipal&accountId=notanaccount",
				json:   nil,
			})

			results := parseResponseArrayJSON(t, resp)
			assert.Equal(t, 0, len(results), "no lease should be returned")
		})

		t.Run("When there is a limit parameter", func(t *testing.T) {
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases?limit=1",
				json:   nil,
			})

			results := parseResponseArrayJSON(t, resp)
			assert.Equal(t, 1, len(results), "only one lease should be returned")
		})

		t.Run("When there is a status parameter", func(t *testing.T) {
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases?status=" + string(db.Inactive),
				json:   nil,
			})

			results := parseResponseArrayJSON(t, resp)
			assert.Equal(t, 2, len(results), "only two leases should be returned")
		})

		t.Run("When there is a Link header", func(t *testing.T) {
			nextPageRegex := regexp.MustCompile(`<(.+)>`)

			respOne := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/leases?limit=2",
				json:   nil,
			})

			linkHeader, ok := respOne.Header["Link"]
			assert.True(t, ok, "Link header should exist")

			resultsOne := parseResponseArrayJSON(t, respOne)
			assert.Equal(t, 2, len(resultsOne), "only two leases should be returned")

			nextPage := nextPageRegex.FindStringSubmatch(linkHeader[0])[1]

			_, err := url.ParseRequestURI(nextPage)
			assert.Nil(t, err, "Link header should contain a valid URL")

			respTwo := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    nextPage,
				json:   nil,
			})

			linkHeader, ok = respTwo.Header["Link"]
			assert.True(t, ok, "Link header should exist")

			resultsTwo := parseResponseArrayJSON(t, respTwo)
			assert.Equal(t, 2, len(resultsTwo), "only two leases should be returned")

			nextPage = nextPageRegex.FindStringSubmatch(linkHeader[0])[1]

			_, err = url.ParseRequestURI(nextPage)
			assert.Nil(t, err, "Link header should contain a valid URL")

			respThree := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    nextPage,
				json:   nil,
			})

			_, ok = respThree.Header["Link"]
			assert.False(t, ok, "Link header should not exist in last page")

			resultsThree := parseResponseArrayJSON(t, respThree)
			assert.Equal(t, 1, len(resultsThree), "only one lease should be returned")

			results := append(resultsOne, resultsTwo...)
			results = append(results, resultsThree...)

			assert.Equal(t, 5, len(results), "All five releases should be returned")
		})
	})

	t.Run("Lease validations", func(t *testing.T) {

		defer truncateAccountTable(t, dbSvc)
		defer truncateLeaseTable(t, dbSvc)

		// Add the current account to the account pool
		apiRequest(t, &apiRequestInput{
			method: "POST",
			url:    apiURL + "/accounts",
			json: createAccountRequest{
				ID:           accountID,
				AdminRoleArn: adminRoleArn,
			},
			maxAttempts: 15,
			f: func(r *testutil.R, apiResp *apiResponse) {
				assert.Equal(r, 201, apiResp.StatusCode)
			},
		})

		// Wait for the account to be reset, so we can lease it
		waitForAccountStatus(t, apiURL, accountID, "Ready")

		t.Run("Should validate requested lease has a desired expiry date less than today", func(t *testing.T) {

			principalID := "user"
			expiresOn := time.Now().AddDate(0, 0, -1).Unix()

			// Create the Provision Request Body
			body := inputLeaseRequest{
				PrincipalID:              principalID,
				BudgetAmount:             200.00,
				ExpiresOn:                expiresOn,
				BudgetCurrency:           "USD",
				BudgetNotificationEmails: []string{"test1@test.com"},
			}

			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "POST",
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
			require.Equal(t, "RequestValidationError", err["code"].(string))
			errStr := fmt.Sprintf("lease validation error: expiresOn: Requested lease has a desired expiry date less than today: %d.", expiresOn)
			require.Equal(t, errStr, err["message"].(string))
		})

		t.Run("Should validate requested budget amount", func(t *testing.T) {

			principalID := "user"
			expiresOn := time.Now().AddDate(0, 0, 5).Unix()

			// Create the Provision Request Body
			body := inputLeaseRequest{
				PrincipalID:              principalID,
				BudgetAmount:             30000.00,
				ExpiresOn:                expiresOn,
				BudgetCurrency:           "USD",
				BudgetNotificationEmails: []string{"test1@test.com"},
			}

			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "POST",
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
			require.Equal(t, "RequestValidationError", err["code"].(string))
			require.Equal(t, "lease validation error: budgetAmount: Requested lease has a budget amount of 30000.000000, which is greater than max lease budget amount of 1000.000000.",
				err["message"].(string))

		})

		t.Run("Should validate requested budget period", func(t *testing.T) {

			principalID := "user"
			expiresOnAfterOneYear := time.Now().AddDate(1, 0, 0).Unix()

			// Create the Provision Request Body

			body := inputLeaseRequest{
				PrincipalID:              principalID,
				BudgetAmount:             300.00,
				ExpiresOn:                expiresOnAfterOneYear,
				BudgetCurrency:           "USD",
				BudgetNotificationEmails: []string{"test1@test.com"},
			}

			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "POST",
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
			errStr := fmt.Sprintf("Requested lease has a budget expires on of %d, which is greater than max lease period of", expiresOnAfterOneYear)
			require.Equal(t, "RequestValidationError", err["code"].(string))
			require.Contains(t, err["message"].(string), errStr)

		})

		t.Run("Should validate requested budget amount against principal budget amount", func(t *testing.T) {
			truncateUsageTable(t, usageSvc)
			defer truncateUsageTable(t, usageSvc)
			createUsage(t, apiURL, usageSvc)

			principalID := "TestUser1"
			expiresOn := time.Now().AddDate(0, 0, 6).Unix()

			// Create the Provision Request Body
			body := inputLeaseRequest{
				PrincipalID:              principalID,
				BudgetAmount:             430.00,
				ExpiresOn:                expiresOn,
				BudgetCurrency:           "USD",
				BudgetNotificationEmails: []string{"test1@test.com"},
			}

			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "POST",
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
			require.Equal(t, "RequestValidationError", err["code"].(string))
			// Weekday + 1 since Sunday is 0.  Min of 5 because thats what the write usage does
			weekday := math.Min(float64(time.Now().Weekday())+1, 5)
			require.Equal(t,
				fmt.Sprintf("lease validation error: budgetAmount: Unable to create lease: User principal TestUser1 "+
					"has already spent %.2f of their 1000.00 principal budget.", weekday*2000),
				err["message"].(string),
			)
		})

	})

	t.Run("Get Accounts", func(t *testing.T) {

		t.Run("should return empty for no accounts", func(t *testing.T) {
			defer truncateAccountTable(t, dbSvc)

			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/accounts",
				json:   nil,
			})

			results := parseResponseArrayJSON(t, resp)

			assert.Equal(t, results, []map[string]interface{}{}, "API should return []")
		})

		truncateAccountTable(t, dbSvc)

		accountIDOne := "1"
		accountIDTwo := "2"
		accountIDThree := "3"
		accountIDFour := "4"
		accountIDFive := "5"

		err = dbSvc.PutAccount(db.Account{
			ID:            accountIDOne,
			AccountStatus: db.Ready,
		})
		assert.Nil(t, err)

		err = dbSvc.PutAccount(db.Account{
			ID:            accountIDTwo,
			AccountStatus: db.Ready,
		})
		assert.Nil(t, err)

		err = dbSvc.PutAccount(db.Account{
			ID:            accountIDThree,
			AccountStatus: db.Ready,
		})
		assert.Nil(t, err)

		err = dbSvc.PutAccount(db.Account{
			ID:            accountIDFour,
			AccountStatus: db.Ready,
		})
		assert.Nil(t, err)

		err = dbSvc.PutAccount(db.Account{
			ID:            accountIDFive,
			AccountStatus: db.NotReady,
		})
		assert.Nil(t, err)

		t.Run("When there are no query parameters", func(t *testing.T) {
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/accounts",
				json:   nil,
			})

			results := parseResponseArrayJSON(t, resp)
			assert.Equal(t, 5, len(results), "all five accounts should be returned")

			// Check one of the result objects, to make sure it looks right
			_, hasAccountID := results[0]["id"]
			_, hasAccountStatus := results[0]["accountStatus"]

			assert.True(t, hasAccountID, "response should be serialized with the accountId property")
			assert.True(t, hasAccountStatus, "response should be serialized with the accountStatus property")
		})

		t.Run("When there is an account ID parameter", func(t *testing.T) {
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/accounts?id=" + accountIDOne,
				json:   nil,
			})

			results := parseResponseArrayJSON(t, resp)
			assert.Equal(t, 1, len(results), "one account should be returned")
		})

		t.Run("When there is a limit parameter", func(t *testing.T) {
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/accounts?limit=1",
				json:   nil,
			})

			results := parseResponseArrayJSON(t, resp)
			assert.Equal(t, 1, len(results), "only one account should be returned")
		})

		t.Run("When there is a status parameter", func(t *testing.T) {
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/accounts?status=" + string(db.NotReady),
				json:   nil,
			})

			results := parseResponseArrayJSON(t, resp)
			assert.Equal(t, 1, len(results), "only one account should be returned")
		})

		t.Run("When there is a Link header", func(t *testing.T) {
			nextPageRegex := regexp.MustCompile(`<(.+)>`)

			respOne := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/accounts?limit=2",
				json:   nil,
			})

			linkHeader, ok := respOne.Header["Link"]
			assert.True(t, ok, "Link header should exist")

			resultsOne := parseResponseArrayJSON(t, respOne)
			assert.Equal(t, 2, len(resultsOne), "only two accounts should be returned")

			nextPage := nextPageRegex.FindStringSubmatch(linkHeader[0])[1]

			_, err := url.ParseRequestURI(nextPage)
			assert.Nil(t, err, "Link header should contain a valid URL")

			respTwo := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    nextPage,
				json:   nil,
			})

			linkHeader, ok = respTwo.Header["Link"]
			assert.True(t, ok, "Link header should exist")

			resultsTwo := parseResponseArrayJSON(t, respTwo)
			assert.Equal(t, 2, len(resultsTwo), "only two accounts should be returned")

			nextPage = nextPageRegex.FindStringSubmatch(linkHeader[0])[1]

			_, err = url.ParseRequestURI(nextPage)
			assert.Nil(t, err, "Link header should contain a valid URL")

			respThree := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    nextPage,
				json:   nil,
			})

			_, ok = respThree.Header["Link"]
			assert.False(t, ok, "Link header should not exist in last page")

			resultsThree := parseResponseArrayJSON(t, respThree)
			assert.Equal(t, 1, len(resultsThree), "only one account should be returned")

			results := append(resultsOne, resultsTwo...)
			results = append(results, resultsThree...)

			assert.Equal(t, 5, len(results), "All six accounts should be returned")
		})
	})
}

type leaseRequest struct {
	PrincipalID string `json:"principalId"`
	AccountID   string `json:"accountId"`
}

type inputLeaseRequest struct {
	PrincipalID              string   `json:"principalId"`
	AccountID                string   `json:"accountId"`
	BudgetAmount             float64  `json:"budgetAmount"`
	ExpiresOn                int64    `json:"expiresOn"`
	BudgetCurrency           string   `json:"budgetCurrency"`
	BudgetNotificationEmails []string `json:"budgetNotificationEmails"`
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
	//		f: func(r *testutil.R, apiResp *apiResponse) {
	//			assert.Equal(r, 200, apiResp.StatusCode)
	//		},
	// or:
	//		f: statusCodeAssertion(200)
	//
	// By default, this will check that the API returns a 2XX response
	f func(r *testutil.R, apiResp *apiResponse)
}

func statusCodeAssertion(statusCode int) func(r *testutil.R, apiResp *apiResponse) {
	return func(r *testutil.R, apiResp *apiResponse) {
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
	testutil.Retry(t, input.maxAttempts, 2*time.Second, func(r *testutil.R) {
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

		// Parse the JSON response
		apiResp = &apiResponse{
			Response: *resp,
		}
		defer resp.Body.Close()
		var data interface{}

		body, err := ioutil.ReadAll(resp.Body)
		assert.NoError(r, err)

		err = json.Unmarshal([]byte(body), &data)
		if err == nil {
			apiResp.json = data
		}

		if input.f != nil {
			input.f(r, apiResp)
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
	testutil.Retry(t, 30, time.Second, func(r *testutil.R) {
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

	testutil.Retry(t, 10, 10*time.Millisecond, func(r *testutil.R) {

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
	testutil.Retry(t, 10, 2*time.Second, func(r *testutil.R) {

		creds = stscreds.NewCredentials(awsSession, roleArn)
		assert.NotNil(r, creds)
	})
	return creds
}

func deleteAdminRole(t *testing.T, role string, policies []string) {
	awsSession, _ := session.NewSession()
	iamSvc := iam.New(awsSession)
	testutil.Retry(t, 10, 2*time.Second, func(r *testutil.R) {
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
	testutil.Retry(t, 10, 2*time.Second, func(r *testutil.R) {
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
		maxAttempts: 120,
		f: func(r *testutil.R, res *apiResponse) {
			assert.Equalf(r, 200, res.StatusCode, "%v", res.json)

			actualStatus := responseJSONString(t, res, "accountStatus")

			// These status changes can take a while. Log output,
			// so we know our tests aren't stuck
			if r.Attempt == 1 || r.Attempt%5 == 0 {
				log.Printf("Waiting for account to be %s. Account is %s", expectedStatus, actualStatus)
			}
			assert.Equalf(r, expectedStatus, actualStatus,
				"Expected account status to change to %s", expectedStatus)
		},
	})

	// Fail now if the status change never happened
	actualStatus := responseJSONString(t, res, "accountStatus")
	require.Equalf(t, expectedStatus, actualStatus,
		"Expected account status to change from %s to %s", actualStatus, expectedStatus)

	time.Sleep(time.Second * 5)

	return res
}
