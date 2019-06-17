package tests

import (
	"encoding/json"
	"fmt"
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
)

func TestApi(t *testing.T) {
	// Grab the API url from Terraform output
	tfOpts := &terraform.Options{
		TerraformDir: "../../modules",
	}
	apiURL := terraform.Output(t, tfOpts, "api_url")
	require.NotNil(t, apiURL)

	t.Run("Authentication", func(t *testing.T) {

		t.Run("should forbid unauthenticated requests", func(t *testing.T) {
			// Send request to the /status API
			resp, err := http.Get(apiURL + "/leases")
			require.Nil(t, err)

			// Should receive a 403
			require.Equal(t, 403, resp.StatusCode, "should return a 403")

			// Parse response body
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

			// Our Lambda is currently returning a 502, if
			// it's missing a valid JWT. That will change, as we remove JWT,
			// at which point this test should be updated.
			// But, we've proven that we've invoked the Lambda
			require.Equal(t, 502, resp.StatusCode)

			// Parse response body
			data := parseResponseJSON(t, resp)

			// Our Lambda is currently returning an error, if
			// it's missing a valid JWT. That will change, as we remove JWT,
			// at which point this test should be updated.
			// But, we've proven that we've invoked the Lambda
			require.Equal(t, "Internal server error", data["message"].(string))
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

			require.NotEqual(t, 403, resp.StatusCode, "Should not return an IAM authorization error")
		})

	})
}

type apiRequestInput struct {
	method string
	url    string
	creds  *credentials.Credentials
	region string
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
	req, err := http.NewRequest("GET", input.url, nil)
	require.Nil(t, err)

	// Sign our API request, using sigv4
	// See https://docs.aws.amazon.com/general/latest/gr/sigv4_signing.html
	signer := sigv4.NewSigner(input.creds)
	now := time.Now().Add(time.Duration(30) * time.Second)
	//time.Sleep(5 * time.Second)
	signedHeaders, err := signer.Sign(req, nil, "execute-api", input.region, now)
	require.Nil(t, err)
	require.NotNil(t, signedHeaders)

	// Send the API requests
	resp, err := http.DefaultClient.Do(req)
	require.Nil(t, err)

	return resp
}

func parseResponseJSON(t *testing.T, resp *http.Response) map[string]interface{} {
	defer resp.Body.Close()
	var data map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&data)
	require.Nil(t, err)

	return data
}
