package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
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

	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/usage"
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
		tfOut["redbox_lease_db_table_name"].(string),
	)

	// Configure the usage service
	usageSvc := usage.New(
		dynamodb.New(
			awsSession,
			aws.NewConfig().WithRegion(tfOut["aws_region"].(string)),
		),
		tfOut["usage_cache_table_name"].(string),
	)

	// Cleanup tables, to start out
	truncateAccountTable(t, dbSvc)
	truncateLeaseTable(t, dbSvc)

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
			defer truncateLeaseTable(t, dbSvc)

			// Create an Account Entry
			acctID := "123"
			principalID := "user"
			timeNow := time.Now().Unix()
			err := dbSvc.PutAccount(db.RedboxAccount{
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
			})

			// Verify response code
			require.Equal(t, http.StatusOK, resp.StatusCode)

			// Parse response json
			data = parseResponseJSON(t, resp)

			// Verify provisioned response json
			require.Equal(t, principalID, data["principalId"].(string))
			require.Equal(t, acctID, data["accountId"].(string))
			require.Equal(t, string(db.Decommissioned),
				data["leaseStatus"].(string))
			require.NotNil(t, data["createdOn"])
			require.NotNil(t, data["lastModifiedOn"])
			require.NotNil(t, data["leaseStatusModifiedOn"])

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
			principalID := "user"
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
			defer truncateLeaseTable(t, dbSvc)

			// Create an Account Entry
			acctID := "123"
			principalID := "user"
			timeNow := time.Now().Unix()
			err := dbSvc.PutAccount(db.RedboxAccount{
				ID:             acctID,
				AccountStatus:  db.Leased,
				LastModifiedOn: timeNow,
			})
			require.Nil(t, err)

			// Create an Lease Entry
			_, err = dbSvc.PutLease(db.RedboxLease{
				PrincipalID:           principalID,
				AccountID:             acctID,
				LeaseStatus:           db.Active,
				CreatedOn:             timeNow,
				LastModifiedOn:        timeNow,
				LeaseStatusModifiedOn: timeNow,
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
			require.Equal(t, http.StatusConflict, resp.StatusCode)

			// Parse response json
			data := parseResponseJSON(t, resp)

			// Verify error response json
			// Get nested json in response json
			errResp := data["error"].(map[string]interface{})
			require.Equal(t, "ClientError", errResp["code"].(string))
			require.Equal(t, "Principal already has an existing Redbox: 123",
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

		t.Run("Should not be able to decommission with no leases", func(t *testing.T) {
			// Create the Provision Request Body
			principalID := "user"
			acctID := "123"
			body := leaseRequest{
				PrincipalID: principalID,
				AccountID:   acctID,
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
			require.Equal(t, "No account leases found for user",
				err["message"].(string))
		})

		t.Run("Should not be able to decommission with wrong account", func(t *testing.T) {
			// Create an Account Entry
			acctID := "123"
			principalID := "user"
			timeNow := time.Now().Unix()
			err := dbSvc.PutAccount(db.RedboxAccount{
				ID:             acctID,
				AccountStatus:  db.Leased,
				LastModifiedOn: timeNow,
			})
			require.Nil(t, err)

			// Create an Lease Entry
			_, err = dbSvc.PutLease(db.RedboxLease{
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
			require.Equal(t, "No active account leases found for user",
				errResp["message"].(string))
		})

		t.Run("Should not be able to decommission with decommissioned account", func(t *testing.T) {
			// Create an Account Entry
			acctID := "123"
			principalID := "user"
			timeNow := time.Now().Unix()
			err := dbSvc.PutAccount(db.RedboxAccount{
				ID:             acctID,
				AccountStatus:  db.NotReady,
				LastModifiedOn: timeNow,
			})
			require.Nil(t, err)

			// Create an Lease Entry
			_, err = dbSvc.PutLease(db.RedboxLease{
				PrincipalID:           principalID,
				AccountID:             acctID,
				LeaseStatus:           db.Decommissioned,
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
			require.Equal(t, "Account Lease is not active for user - 123",
				errResp["message"].(string))
		})

	})

	t.Run("Account Creation Deletion Flow", func(t *testing.T) {
		// Make sure the DB is clean
		truncateDBTables(t, dbSvc)

		// Create an adminRole for the account
		adminRoleRes := createAdminRole(t, awsSession)
		accountID := adminRoleRes.accountID
		adminRoleArn := adminRoleRes.adminRoleArn

		// Cleanup the DB when we'ree done
		defer truncateDBTables(t, dbSvc)

		t.Run("STEP: Create Account", func(t *testing.T) {

			// Add the current account to the account pool
			createAccountRes := apiRequest(t, &apiRequestInput{
				method: "POST",
				url:    apiURL + "/accounts",
				json: createAccountRequest{
					ID:           accountID,
					AdminRoleArn: adminRoleArn,
				},
			})

			// Check the response
			require.Equal(t, createAccountRes.StatusCode, 201)
			postResJSON := parseResponseJSON(t, createAccountRes)
			require.Equal(t, accountID, postResJSON["id"])
			require.Equal(t, "NotReady", postResJSON["accountStatus"])
			require.Equal(t, adminRoleArn, postResJSON["adminRoleArn"])
			expectedPrincipalRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, tfOut["redbox_principal_role_name"])
			require.Equal(t, expectedPrincipalRoleArn, postResJSON["principalRoleArn"])
			require.True(t, postResJSON["lastModifiedOn"].(float64) > 1561518000)
			require.True(t, postResJSON["createdOn"].(float64) > 1561518000)

			// Check that the account is added to the DB
			dbAccount, err := dbSvc.GetAccount(accountID)
			require.Nil(t, err)
			require.Equal(t, &db.RedboxAccount{
				ID:                  accountID,
				AccountStatus:       "NotReady",
				LastModifiedOn:      int64(postResJSON["lastModifiedOn"].(float64)),
				CreatedOn:           int64(postResJSON["createdOn"].(float64)),
				AdminRoleArn:        adminRoleArn,
				PrincipalRoleArn:    expectedPrincipalRoleArn,
				PrincipalPolicyHash: "\"76807b34385a7bc4cf758c71071e2697\"",
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
				getRes := apiRequest(t, &apiRequestInput{
					method: "GET",
					url:    apiURL + "/accounts/" + accountID,
				})

				// Check the GET /accounts response
				require.Equal(t, getRes.StatusCode, 200)
				getResJSON := getRes.json.(map[string]interface{})
				require.Equal(t, accountID, getResJSON["id"])
				require.Equal(t, "NotReady", getResJSON["accountStatus"])
				require.Equal(t, adminRoleArn, getResJSON["adminRoleArn"])
				expectedPrincipalRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, tfOut["redbox_principal_role_name"])
				require.Equal(t, expectedPrincipalRoleArn, getResJSON["principalRoleArn"])
				require.True(t, getResJSON["lastModifiedOn"].(float64) > 1561518000)
				require.True(t, getResJSON["createdOn"].(float64) > 1561518000)
			})

			t.Run("STEP: List Accounts", func(t *testing.T) {
				// Send GET /accounts
				listRes := apiRequest(t, &apiRequestInput{
					method: "GET",
					url:    apiURL + "/accounts",
				})

				// Check the response
				require.Equal(t, listRes.StatusCode, 200)
				listResJSON := parseResponseArrayJSON(t, listRes)
				accountJSON := listResJSON[0]
				require.Equal(t, accountID, accountJSON["id"])
				require.Equal(t, "NotReady", accountJSON["accountStatus"])
				require.Equal(t, adminRoleArn, accountJSON["adminRoleArn"])
				expectedPrincipalRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, tfOut["redbox_principal_role_name"])
				require.Equal(t, expectedPrincipalRoleArn, accountJSON["principalRoleArn"])
				require.True(t, accountJSON["lastModifiedOn"].(float64) > 1561518000)
				require.True(t, accountJSON["createdOn"].(float64) > 1561518000)
			})

			t.Run("STEP: Create Lease", func(t *testing.T) {
				// Account is being reset, so it's not marked as "Ready".
				// Update the DB to be ready, so we can create a lease
				_, err := dbSvc.TransitionAccountStatus(accountID, db.NotReady, db.Ready)
				require.Nil(t, err)

				var budgetAmount float64 = 300
				var budgetNotificationEmails = []string{"test@test.com"}

				// Request a lease
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
				})

				require.Equal(t, 201, res.StatusCode)
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
				require.NotNil(t, resJSON["leaseStatusModifiedOn"])

				// Check the lease is in the DB
				// (since we dont' yet have a GET /leases endpoint
				lease, err := dbSvc.GetLease(accountID, "test-user")
				require.Nil(t, err)
				require.Equal(t, "test-user", lease.PrincipalID)
				require.Equal(t, accountID, lease.AccountID)

				t.Run("STEP: Delete Account (with Lease)", func(t *testing.T) {
					// Request a lease
					res := apiRequest(t, &apiRequestInput{
						method: "DELETE",
						url:    apiURL + "/accounts/" + accountID,
						json: struct {
							PrincipalID string `json:"principalId"`
						}{
							PrincipalID: "test-user",
						},
					})

					require.Equal(t, 409, res.StatusCode)
				})

				t.Run("STEP: Delete Lease", func(t *testing.T) {
					// Delete the lease
					res := apiRequest(t, &apiRequestInput{
						method: "DELETE",
						url:    apiURL + "/leases",
						json: struct {
							PrincipalID string `json:"principalId"`
							AccountID   string `json:"accountId"`
						}{
							PrincipalID: "test-user",
							AccountID:   accountID,
						},
					})

					require.Equal(t, 200, res.StatusCode)

					// Check the lease is decommissioned
					// (since we dont' yet have a GET /leases endpoint
					lease, err := dbSvc.GetLease(accountID, "test-user")
					require.Nil(t, err)
					require.Equal(t, db.Decommissioned, lease.LeaseStatus)

					t.Run("STEP: Delete Account", func(t *testing.T) {
						// Delete the account
						res := apiRequest(t, &apiRequestInput{
							method: "DELETE",
							url:    apiURL + "/accounts/" + accountID,
						})
						require.Equal(t, 204, res.StatusCode)

						// Attempt to get the deleted account (should 404)
						getRes := apiRequest(t, &apiRequestInput{
							method: "GET",
							url:    apiURL + "/accounts/" + accountID,
						})
						require.Equal(t, 404, getRes.StatusCode)

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

	t.Run("Delete Account", func(t *testing.T) {

		t.Run("when the account does not exists", func(t *testing.T) {
			resp := apiRequest(t, &apiRequestInput{
				method: "DELETE",
				url:    apiURL + "/accounts/1234523456",
			})

			require.Equal(t, http.StatusNotFound, resp.StatusCode, "it returns a 404")
		})

	})

	t.Run("Get Usage api", func(t *testing.T) {

		t.Run("Should get an error for invalid date format", func(t *testing.T) {

			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/usages?startDate=2019-09-2&endDate=2019-09-2",
				json:   nil,
			})

			// Verify response code
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)

			// Parse response json
			data := parseResponseJSON(t, resp)

			// Verify error response json
			// Get nested json in response json
			errResp := data["error"].(map[string]interface{})
			require.Equal(t, "Invalid startDate", errResp["code"].(string))
			require.Equal(t, "Failed to parse usage start date: parsing time \"2019-09-2\" as \"2006-01-02\": cannot parse \"2\" as \"02\"",
				errResp["message"].(string))
		})

		t.Run("Should get an empty json for usage not found for given input date range", func(t *testing.T) {

			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/usages?startDate=1568937600&endDate=1569023999",
				json:   nil,
			})

			// Verify response code
			require.Equal(t, http.StatusOK, resp.StatusCode)

			// Parse response json
			data := parseResponseArrayJSON(t, resp)

			// Verify response json
			require.Equal(t, []map[string]interface{}([]map[string]interface{}{}), data)
		})

		t.Run("Should be able to get usage", func(t *testing.T) {

			// Create usage
			// Setup usage dates
			const ttl int = 3
			testStartDate := time.Date(2019, 5, 5, 0, 0, 0, 0, time.UTC)
			testEndDate := time.Date(2019, 5, 5, 23, 59, 59, 0, time.UTC)

			// Create mock usages
			expectedUsages := []*usage.Usage{}
			for a := 1; a <= 2; a++ {

				startDate := testStartDate
				endDate := testEndDate

				timeToLive := startDate.AddDate(0, 0, ttl)

				var testPrinciplaID []string
				var testAccountID []string

				testPrinciplaID = append(testPrinciplaID, "TestUser")
				testPrinciplaID = append(testPrinciplaID, strconv.Itoa(a))

				testAccountID = append(testAccountID, "TestAcct")
				testAccountID = append(testAccountID, strconv.Itoa(a))

				for i := 1; i <= 3; i++ {

					input := usage.Usage{
						PrincipalID:  strings.Join(testPrinciplaID, ""),
						AccountID:    strings.Join(testAccountID, ""),
						StartDate:    startDate.Unix(),
						EndDate:      endDate.Unix(),
						CostAmount:   20.00,
						CostCurrency: "USD",
						TimeToLive:   timeToLive.Unix(),
					}
					err = usageSvc.PutUsage(input)
					require.Nil(t, err)
					expectedUsages = append(expectedUsages, &input)

					startDate = startDate.AddDate(0, 0, 1)
					endDate = endDate.AddDate(0, 0, 1)
				}
			}

			// Send an API request
			resp := apiRequest(t, &apiRequestInput{
				method: "GET",
				url:    apiURL + "/usages?startDate=1557014400&endDate=1557273599",
				json:   nil,
			})

			// Verify response code
			require.Equal(t, http.StatusOK, resp.StatusCode)

			// Parse response json
			data := parseResponseArrayJSON(t, resp)
			fmt.Printf("data : %v", data)

			//Verify response json
			if data[0] != nil {
				usageJSON := data[0]
				require.Equal(t, "TestUser1", usageJSON["principalId"].(string))
				require.Equal(t, "TestAcct1", usageJSON["accountId"].(string))
				require.Equal(t, 60.00, usageJSON["costAmount"].(float64))
			}
		})
	})
}

type leaseRequest struct {
	PrincipalID string `json:"principalId"`
	AccountID   string `json:"accountId"`
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

type apiResponse struct {
	http.Response
	json interface{}
}

func apiRequest(t *testing.T, input *apiRequestInput) *apiResponse {
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

	// Parse the JSON response
	apiResp := &apiResponse{
		Response: *resp,
	}
	defer resp.Body.Close()
	var data interface{}

	body, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)

	err = json.Unmarshal([]byte(body), &data)
	if err == nil {
		apiResp.json = data
	}

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
	adminRoleArn string
}

func createAdminRole(t *testing.T, awsSession client.ConfigProvider) *createAdminRoleOutput {
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
	adminRoleName := "redbox-api-test-admin-role-" + fmt.Sprintf("%v", time.Now().Unix())
	roleRes, err := iamSvc.CreateRole(&iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(assumeRolePolicy),
		Path:                     aws.String("/"),
		RoleName:                 aws.String(adminRoleName),
	})
	require.Nil(t, err)
	adminRoleArn := *roleRes.Role.Arn

	// Give the Admin Role Permission to create other IAM Roles
	// (so it can create a role for the Redbox principal)
	_, err = iamSvc.AttachRolePolicy(&iam.AttachRolePolicyInput{
		RoleName:  aws.String(adminRoleName),
		PolicyArn: aws.String("arn:aws:iam::aws:policy/IAMFullAccess"),
	})
	require.Nil(t, err)

	// IAM Role takes a while to propagate....
	time.Sleep(10 * time.Second)

	return &createAdminRoleOutput{
		adminRoleArn: adminRoleArn,
		accountID:    currentAccountID,
	}
}
