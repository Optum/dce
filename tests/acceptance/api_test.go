package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

var adminRoleName = ""

func TestMain(m *testing.M) {
	code := m.Run()
	deleteAdminRole()
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
	adminRoleName = "dce-api-test-admin-role-" + fmt.Sprintf("%v", time.Now().Unix())
	adminRoleRes := createAdminRole(t, awsSession, adminRoleName)
	accountID := adminRoleRes.accountID
	adminRoleArn := adminRoleRes.adminRoleArn

	// Cleanup tables before and after tests
	truncateAccountTable(t, dbSvc)
	truncateLeaseTable(t, dbSvc)
	truncateUsageTable(t, usageSvc)
	defer truncateAccountTable(t, dbSvc)
	defer truncateLeaseTable(t, dbSvc)
	defer truncateUsageTable(t, usageSvc)

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

		t.Run("Should be able to create and destroy and lease", func(t *testing.T) {
			defer truncateAccountTable(t, dbSvc)
			defer truncateLeaseTable(t, dbSvc)

			// Create an Account Entry
			acctID := "123"
			principalID := "user"
			timeNow := time.Now().Unix()
			err := dbSvc.PutAccount(db.Account{
				ID:             acctID,
				AccountStatus:  db.Ready,
				LastModifiedOn: timeNow,
			})
			require.Nil(t, err)

			// Create the Provision Request Body
			body := leaseRequest{
				PrincipalID: principalID,
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
			require.Equal(t, principalID, data["principalId"].(string))
			require.Equal(t, acctID, data["accountId"].(string))
			require.Equal(t, string(db.Active),
				data["leaseStatus"].(string))
			require.NotNil(t, data["createdOn"])
			require.NotNil(t, data["lastModifiedOn"])
			require.NotNil(t, data["leaseStatusModifiedOn"])

			// Create the Decommission Request Body
			body = leaseRequest{
				PrincipalID: principalID,
				AccountID:   acctID,
			}

			// Send an API request
			resp = apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + "/leases",
				json:   body,
				f: func(r *testutil.R, apiResp *apiResponse) {
					// Verify response code
					assert.Equal(r, http.StatusOK, apiResp.StatusCode)
				},
			})

			// Parse response json
			data = parseResponseJSON(t, resp)

			// Verify provisioned response json
			assert.Equal(t, principalID, data["principalId"].(string))
			assert.Equal(t, acctID, data["accountId"].(string))
			assert.Equal(t, string(db.Inactive),
				data["leaseStatus"].(string))
			assert.NotNil(t, data["createdOn"])
			assert.NotNil(t, data["lastModifiedOn"])
			assert.NotNil(t, data["leaseStatusModifiedOn"])

		})

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
					assert.Equal(r, "RequestValidationError", err["code"].(string))
					assert.Equal(r, "invalid request parameters",
						err["message"].(string))
				},
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
					assert.Equal(r, http.StatusServiceUnavailable, apiResp.StatusCode)

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

		t.Run("Should not be able to create lease if there's already an existing lease for the principal", func(t *testing.T) {
			defer truncateAccountTable(t, dbSvc)
			defer truncateLeaseTable(t, dbSvc)

			// Create an Account Entry
			acctID := "123"
			principalID := "user"
			timeNow := time.Now().Unix()
			err := dbSvc.PutAccount(db.Account{
				ID:             acctID,
				AccountStatus:  db.Leased,
				LastModifiedOn: timeNow,
			})
			require.Nil(t, err)

			// Create an Lease Entry
			_, err = dbSvc.PutLease(db.Lease{
				ID:                    uuid.New().String(),
				PrincipalID:           principalID,
				AccountID:             acctID,
				LeaseStatus:           db.Active,
				CreatedOn:             timeNow,
				LastModifiedOn:        timeNow,
				LeaseStatusModifiedOn: timeNow,
			})
			require.Nil(t, err)

			// Create the lease Request Body
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
					assert.Equal(r, http.StatusConflict, apiResp.StatusCode)

					// Parse response json
					data := parseResponseJSON(t, apiResp)

					// Verify error response json
					// Get nested json in response json
					errResp := data["error"].(map[string]interface{})
					assert.Equal(r, "ClientError", errResp["code"].(string))
					assert.Equal(r, "Principal already has an active lease for account 123",
						errResp["message"].(string))
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
					assert.Equal(r, "Failed to Parse Request Body: ",
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
					assert.Equal(r, http.StatusBadRequest, apiResp.StatusCode)

					// Parse response json
					data := parseResponseJSON(t, apiResp)

					// Verify error response json
					// Get nested json in response json
					err := data["error"].(map[string]interface{})
					assert.Equal(r, "ClientError", err["code"].(string))
					assert.Equal(r, "No leases found for user",
						err["message"].(string))
				},
			})

		})

		t.Run("Should not be able to destroy lease with wrong account", func(t *testing.T) {
			// Create an Account Entry
			acctID := "123"
			principalID := "user"
			timeNow := time.Now().Unix()
			err := dbSvc.PutAccount(db.Account{
				ID:             acctID,
				AccountStatus:  db.Leased,
				LastModifiedOn: timeNow,
			})
			require.Nil(t, err)

			// Create an Lease Entry
			_, err = dbSvc.PutLease(db.Lease{
				ID:                    uuid.New().String(),
				PrincipalID:           principalID,
				AccountID:             acctID,
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
					assert.Equal(r, http.StatusBadRequest, apiResp.StatusCode)

					// Parse response json
					data := parseResponseJSON(t, apiResp)

					// Verify error response json
					// Get nested json in response json
					errResp := data["error"].(map[string]interface{})
					assert.Equal(r, "ClientError", errResp["code"].(string))
					assert.Equal(r, "No active leases found for user",
						errResp["message"].(string))
				},
			})

		})

		t.Run("Should not be able to destroy lease with NotReady account", func(t *testing.T) {
			// Create an Account Entry
			acctID := "123"
			principalID := "user"
			timeNow := time.Now().Unix()
			err := dbSvc.PutAccount(db.Account{
				ID:             acctID,
				AccountStatus:  db.NotReady,
				LastModifiedOn: timeNow,
			})
			require.Nil(t, err)

			// Create an Lease Entry
			_, err = dbSvc.PutLease(db.Lease{
				ID:                    uuid.New().String(),
				PrincipalID:           principalID,
				AccountID:             acctID,
				LeaseStatus:           db.Inactive,
				CreatedOn:             timeNow,
				LastModifiedOn:        timeNow,
				LeaseStatusModifiedOn: timeNow,
			})
			require.Nil(t, err)

			// Create the Provision Request Body
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
					assert.Equal(r, http.StatusBadRequest, apiResp.StatusCode)

					// Parse response json
					data := parseResponseJSON(t, apiResp)

					// Verify error response json
					// Get nested json in response json
					errResp := data["error"].(map[string]interface{})
					assert.Equal(r, "ClientError", errResp["code"].(string))
					assert.Equal(r, "Lease is not active for user - 123",
						errResp["message"].(string))
				},
			})

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
				// Account is being reset, so it's not marked as "Ready".
				// Update the DB to be ready, so we can create a lease
				_, err := dbSvc.TransitionAccountStatus(accountID, db.NotReady, db.Ready)
				require.Nil(t, err)

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

				// Check the lease is in the DB
				// (since we dont' yet have a GET /leases endpoint
				lease, err := dbSvc.GetLease(accountID, "test-user")
				require.Nil(t, err)
				require.Equal(t, "test-user", lease.PrincipalID)
				require.Equal(t, accountID, lease.AccountID)

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
					// (since we dont' yet have a GET /leases endpoint
					lease, err := dbSvc.GetLease(accountID, "test-user")
					require.Nil(t, err)
					require.Equal(t, db.Inactive, lease.LeaseStatus)

					t.Run("STEP: Recreate lease against same account", func(t *testing.T) {
						// Account is being reset, so it's not marked as "Ready".
						// Update the DB to be ready, so we can create a lease
						_, err := dbSvc.TransitionAccountStatus(accountID, db.NotReady, db.Ready)
						require.Nil(t, err)

						// Request a lease
						// Because we only have one account in our system,
						// this will create a lease against the same account.
						// We want to make sure that the lease parameters get updated.
						time.Sleep(1 * time.Second) // Wait a bit, so our timestamps are new
						expiresOn := time.Now().Unix() + 1000
						res = apiRequest(t, &apiRequestInput{
							method: "POST",
							url:    apiURL + "/leases",
							json: map[string]interface{}{
								"principalId": "test-user",
								// Change some values here, compared to the previous lease
								"budgetAmount":   500,
								"budgetCurrency": "EUR",
								"expiresOn":      expiresOn,
							},
							f: func(r *testutil.R, apiResp *apiResponse) {
								assert.Equal(r, 201, apiResp.StatusCode)
							},
						})
						resJSON = parseResponseJSON(t, res)

						// Check values in JSON response are updated
						require.Equal(t, float64(500), resJSON["budgetAmount"])
						require.Equal(t, "EUR", resJSON["budgetCurrency"])
						require.Equal(t, float64(expiresOn), resJSON["expiresOn"])
						// Check that our timestamps are updated
						require.True(t, resJSON["createdOn"].(float64) > float64(lease.CreatedOn))
						require.True(t, resJSON["lastModifiedOn"].(float64) > float64(lease.LastModifiedOn))
						require.True(t, resJSON["leaseStatusModifiedOn"].(float64) > float64(lease.LeaseStatusModifiedOn))

						// Lookup the lease in the DB, to make sure it was updated
						newLease, err := dbSvc.GetLeaseByID(resJSON["id"].(string))
						require.Nil(t, err)
						require.Equal(t, float64(500), newLease.BudgetAmount)
						require.Equal(t, "EUR", newLease.BudgetCurrency)
						require.Equal(t, expiresOn, newLease.ExpiresOn)
						// Check that our timestamps are updated
						require.True(t, newLease.CreatedOn > lease.CreatedOn)
						require.True(t, newLease.LastModifiedOn > lease.LastModifiedOn)
						require.True(t, newLease.LeaseStatusModifiedOn > lease.LeaseStatusModifiedOn)

						// Delete the lease (cleanup)
						apiRequest(t, &apiRequestInput{
							method: "DELETE",
							url:    apiURL + "/leases",
							json: map[string]interface{}{
								"principalId": "test-user",
								"accountId":   accountID,
							},
							f: func(r *testutil.R, apiResp *apiResponse) {
								assert.Equal(r, 200, apiResp.StatusCode)
							},
						})
					})

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
					// this shouldn't actually persist
					"accountStatus": "hippos",
				},
				f: statusCodeAssertion(200),
			})

			// Check the JSON response
			resJSON := parseResponseJSON(t, res)
			require.Equal(t, map[string]interface{}{
				"foo": "bar",
			}, resJSON["metadata"], "Response includes updated metadata")
			require.NotEqual(t, "hippos", resJSON["accountStatus"],
				"shouldn't update non-updatable fields")
			require.True(t, resJSON["lastModifiedOn"].(float64) > resJSON["createdOn"].(float64),
				"should update lastModifiedOn timestamp")

			// Check the DB record, to make sure it's updated
			account, err := dbSvc.GetAccount(accountID)
			require.Nil(t, err)

			require.Equal(t, map[string]interface{}{
				"foo": "bar",
			}, account.Metadata, "db record metadata is updated")
			require.NotEqual(t, "hippos", account.AccountStatus,
				"shouldn't update non-updatable fields in DB")
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
					"code": "RequestValidationError",
					"message": fmt.Sprintf("Unable to update account %s: "+
						"admin role is not assumable by the master account", accountID),
				},
			}, resJSON)
		})

		t.Run("should return a 404 if the account doesn't exist", func(t *testing.T) {
			// PUT /accounts/:id
			// with invalid adminRoleArn
			res := apiRequest(t, &apiRequestInput{
				method: "PUT",
				url:    apiURL + "/accounts/not-an-account-id",
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
					"code":    "NotFound",
					"message": "The requested resource could not be found.",
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
						assert.Equal(r, "TestAccount1", usageJSON["accountId"].(string))
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
						assert.Equal(r, "TestAccount1", usageJSON["accountId"].(string))
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
						assert.Equal(r, "TestAccount1", usageJSON["accountId"].(string))
						assert.Equal(r, 10000.00, usageJSON["costAmount"].(float64))
					}
				})
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

			linkHeader, ok = respThree.Header["Link"]
			assert.False(t, ok, "Link header should not exist in last page")

			resultsThree := parseResponseArrayJSON(t, respThree)
			assert.Equal(t, 1, len(resultsThree), "only one lease should be returned")

			results := append(resultsOne, resultsTwo...)
			results = append(results, resultsThree...)

			assert.Equal(t, 5, len(results), "All five releases should be returned")
		})
	})

	t.Run("Lease validations", func(t *testing.T) {

		t.Run("Should validate requested lease has a desired expiry date less than today", func(t *testing.T) {

			principalID := "user"
			expiresOn := time.Now().AddDate(0, 0, -1).Unix()

			// Create the Provision Request Body
			body := inputLeaseRequest{
				PrincipalID:  principalID,
				AccountID:    "123",
				BudgetAmount: 200.00,
				ExpiresOn:    expiresOn,
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
			errStr := fmt.Sprintf("Requested lease has a desired expiry date less than today: %d", expiresOn)
			require.Equal(t, errStr, err["message"].(string))
		})

		t.Run("Should validate requested budget amount", func(t *testing.T) {

			principalID := "user"
			expiresOn := time.Now().AddDate(0, 0, 5).Unix()

			// Create the Provision Request Body
			body := inputLeaseRequest{
				PrincipalID:  principalID,
				AccountID:    "123",
				BudgetAmount: 30000.00,
				ExpiresOn:    expiresOn,
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
			require.Equal(t, "Requested lease has a budget amount of 30000.000000, which is greater than max lease budget amount of 1000.000000",
				err["message"].(string))

		})

		t.Run("Should validate requested budget period", func(t *testing.T) {

			principalID := "user"
			expiresOnAfterOneYear := time.Now().AddDate(1, 0, 0).Unix()

			// Create the Provision Request Body
			body := inputLeaseRequest{
				PrincipalID:  principalID,
				AccountID:    "123",
				BudgetAmount: 300.00,
				ExpiresOn:    expiresOnAfterOneYear,
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

			defer truncateUsageTable(t, usageSvc)
			createUsage(t, apiURL, usageSvc)

			principalID := "TestUser1"
			expiresOn := time.Now().AddDate(0, 0, 6).Unix()

			// Create the Provision Request Body
			body := inputLeaseRequest{
				PrincipalID:  principalID,
				AccountID:    "123",
				BudgetAmount: 430.00,
				ExpiresOn:    expiresOn,
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
			errStr := fmt.Sprintf("Unable to create lease: User principal %s has already spent 1000.000000 of their principal budget", principalID)
			require.Equal(t, "RequestValidationError", err["code"].(string))
			require.Equal(t, errStr, err["message"].(string))
		})

	})
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
		input.maxAttempts = 15
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

func parseResponseJSON(t *testing.T, resp *apiResponse) map[string]interface{} {
	require.NotNil(t, resp.json)
	return resp.json.(map[string]interface{})
}

func parseResponseArrayJSON(t *testing.T, resp *apiResponse) []map[string]interface{} {
	require.NotNil(t, resp.json)

	// Go doesn't allow you to cast directly to []map[string]interface{}
	// so we need to mess around here a bit.
	// This might be relevant: https://stackoverflow.com/questions/38579485/golang-convert-slices-into-map
	respJSON := resp.json.([]interface{})

	arrJSON := []map[string]interface{}{}
	for _, obj := range respJSON {
		arrJSON = append(arrJSON, obj.(map[string]interface{}))
	}

	return arrJSON
}

type createAdminRoleOutput struct {
	accountID    string
	roleName     string
	adminRoleArn string
}

func createAdminRole(t *testing.T, awsSession client.ConfigProvider, adminRoleName string) *createAdminRoleOutput {
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

	// Give the Admin Role Permission to create other IAM Roles
	// (so it can create a role for the principal)
	_, err = iamSvc.AttachRolePolicy(&iam.AttachRolePolicyInput{
		RoleName:  aws.String(adminRoleName),
		PolicyArn: aws.String("arn:aws:iam::aws:policy/IAMFullAccess"),
	})

	// Give the Admin Role Permission to access cost explorer
	_, err = iamSvc.AttachRolePolicy(&iam.AttachRolePolicyInput{
		RoleName:  aws.String(adminRoleName),
		PolicyArn: aws.String("arn:aws:iam::391501768339:policy/CostExplorerFullAccess"),
	})
	require.Nil(t, err)

	// IAM Role takes a while to propagate....
	//time.Sleep(10 * time.Second)

	return &createAdminRoleOutput{
		adminRoleArn: adminRoleArn,
		roleName:     adminRoleName,
		accountID:    currentAccountID,
	}
}

func createUsage(t *testing.T, apiURL string, usageSvc usage.Service) {
	// Create usage
	// Setup usage dates
	const ttl int = 3
	currentDate := time.Now()
	testStartDate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, time.UTC)
	testEndDate := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 23, 59, 59, 59, time.UTC)

	// Create mock usage
	var expectedUsages []*usage.Usage

	usageStartDate := testStartDate
	usageEndDate := testEndDate
	startDate := testStartDate
	endDate := testEndDate

	timeToLive := startDate.AddDate(0, 0, ttl)

	var testPrincipalID = "TestUser1"
	var testAccountID = "TestAccount1"

	for i := 1; i <= 5; i++ {

		input := usage.Usage{
			PrincipalID:  testPrincipalID,
			AccountID:    testAccountID,
			StartDate:    startDate.Unix(),
			EndDate:      endDate.Unix(),
			CostAmount:   2000.00,
			CostCurrency: "USD",
			TimeToLive:   timeToLive.Unix(),
		}
		err := usageSvc.PutUsage(input)
		require.Nil(t, err)

		expectedUsages = append(expectedUsages, &input)

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

func deleteAdminRole() {
	awsSession, _ := session.NewSession()
	iamSvc := iam.New(awsSession)
	_, _ = iamSvc.DeleteRole(&iam.DeleteRoleInput{
		RoleName: aws.String(adminRoleName),
	})
}
