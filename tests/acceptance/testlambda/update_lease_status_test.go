package testlambda

import (
	"bytes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"encoding/json"
	"fmt"

	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/service/dynamodb"

	"github.com/stretchr/testify/assert"

	"github.com/Optum/dce/tests/acceptance/testutil"
	"github.com/aws/aws-sdk-go/aws/credentials"
	sigv4 "github.com/aws/aws-sdk-go/aws/signer/v4"
)

type inputLeaseRequest struct {
	PrincipalID  string  `json:"principalId"`
	AccountID    string  `json:"accountId"`
	BudgetAmount float64 `json:"budgetAmount"`
	ExpiresOn    int64   `json:"expiresOn"`
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
	// eg.
	//		f: func(r *testutil.R, apiResp *apiResponse) {
	//			assert.Equal(r, 200, apiResp.StatusCode)
	//		},
	f func(r *testutil.R, apiResp *apiResponse)
}

type apiResponse struct {
	http.Response
	json interface{}
}

type getItemsRequest struct {
	SortBy      string
	SortOrder   string
	ItemsToGet  int
	PrincipalID string
	AccountID   string
}

type getItemsResponseError struct {
	Message string `json:"message"`
}

type getItemsResponseData struct {
	Item string `json:"item"`
}

type getItemsResponseBody struct {
	Result string                 `json:"result"`
	Data   []getItemsResponseData `json:"data"`
	Error  getItemsResponseError  `json:"error"`
}

type getItemsResponseHeaders struct {
	ContentType string `json:"Content-Type"`
}

type getItemsResponse struct {
	StatusCode int                     `json:"statusCode"`
	Headers    getItemsResponseHeaders `json:"headers"`
	Body       getItemsResponseBody    `json:"body"`
}

func TestUpdateLeaseStatusLambda(t *testing.T) {
	// Load Terraform outputs
	tfOpts := &terraform.Options{
		TerraformDir: "../../../modules",
	}
	tfOut := terraform.OutputAll(t, tfOpts)
	apiURL := tfOut["api_url"].(string)

	// Configure the DB service
	// Configure the S3 Service
	awsSession, err := session.NewSession(
		aws.NewConfig().WithRegion(tfOut["aws_region"].(string)))
	require.Nil(t, err)
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



	// Create Lambda service client
	lambdaClient := lambda.New(awsSession)

	// Test update_lease_status lambda
	t.Run("update_lease_status", func(t *testing.T) {
		t.Run("Should run update_lease_status lambda successfully", func(t *testing.T) {

			// Create an account and Lease
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
			body := inputLeaseRequest{
				PrincipalID: principalID,
			}

			// Send an API request
			leaseResp := apiRequest(t, &apiRequestInput{
				method: "POST",
				url:    apiURL + "/leases",
				json:   body,
			})

			// Verify response code
			require.Equal(t, http.StatusCreated, leaseResp.StatusCode)


			// Get the 10 most recent items
			request := getItemsRequest{"time", "descending", 10, principalID, acctID}

			payload, err := json.Marshal(request)
			require.Nil(t, err)

			result, err := lambdaClient.Invoke(&lambda.InvokeInput{FunctionName: aws.String("update_lease_status-" + tfOut["namespace"].(string)), Payload: payload})
			require.Nil(t, err)

			var resp getItemsResponse

			err = json.Unmarshal(result.Payload, &resp)
			require.Nil(t, err)

			// If the status code is NOT 200, the call failed
			require.Equal(t, http.StatusOK, resp.StatusCode)

			// Print out items
			if len(resp.Body.Data) > 0 {
				for i := range resp.Body.Data {
					fmt.Println(resp.Body.Data[i].Item)
				}
			} else {
				fmt.Println("There were no items")
			}

		})
	})
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


