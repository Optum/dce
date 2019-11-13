package testlambda

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"

	"encoding/json"
	"fmt"
)

type getItemsRequest struct {
	SortBy     string
	SortOrder  string
	ItemsToGet int
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

	// Configure the S3 Service
	awsSession, err := session.NewSession(
		aws.NewConfig().WithRegion(tfOut["aws_region"].(string)))
	require.Nil(t, err)

	// Create Lambda service client
	lambdaClient := lambda.New(awsSession)

	// Test update_lease_status lambda
	t.Run("update_lease_status", func(t *testing.T) {
		t.Run("Should run update_lease_status lambda successfully", func(t *testing.T) {

			// Get the 10 most recent items
			request := getItemsRequest{"time", "descending", 10}

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
