package tests

import (
	"bytes"
	"github.com/aws/aws-sdk-go/aws/credentials"
	sigv4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func TestCredentialsWebPageLoads(t *testing.T) {

	// Load Terraform outputs
	tfOpts := &terraform.Options{
		TerraformDir: "../../modules",
	}
	tfOut := terraform.OutputAll(t, tfOpts)
	apiURL := tfOut["api_url"].(string)

	var chainCredentials = credentials.NewChainCredentials([]credentials.Provider{
		&credentials.EnvProvider{},
		&credentials.SharedCredentialsProvider{Filename: "", Profile: ""},
	})

	endpointUrl := apiURL + "/auth"
	region := "us-east-1"
	creds := chainCredentials
	httpMethod := "GET"

	t.Run("Serves web page with configuration.", func(t *testing.T) {
		// Arrange
		req, err := http.NewRequest(httpMethod, endpointUrl, nil)
		assert.Nil(t, err)

		signRequest(t, req, creds, region)

		httpClient := http.Client{
			Timeout: 60 * time.Second,
		}

		// Act
		resp, err := httpClient.Do(req)
		assert.NoError(t, err)

		// Assert
		assert.Equal(t, 200, resp.StatusCode)
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		assert.NotContains(t, buf.String(), `SITE_PATH_PREFIX = ""`)
		assert.NotContains(t, buf.String(), `APIGW_DEPLOYMENT_NAME = ""`)
		assert.NotContains(t, buf.String(), `AWS_CURRENT_REGION = ""`)
		assert.NotContains(t, buf.String(), `IDENTITY_POOL_ID = ""`)
		assert.NotContains(t, buf.String(), `USER_POOL_PROVIDER_NAME = ""`)
		assert.NotContains(t, buf.String(), `USER_POOL_CLIENT_ID = ""`)
		assert.NotContains(t, buf.String(), `USER_POOL_APP_WEB_DOMAIN = ""`)
		assert.NotContains(t, buf.String(), `USER_POOL_ID = ""`)
	})

	t.Run("Serves static assets.", func(t *testing.T) {
		// Arrange
		staticAssetUrl := endpointUrl + "/public/main.css"
		req, err := http.NewRequest(httpMethod, staticAssetUrl, nil)
		assert.Nil(t, err)

		signRequest(t, req, creds, region)

		httpClient := http.Client{
			Timeout: 60 * time.Second,
		}

		// Act
		resp, err := httpClient.Do(req)
		assert.NoError(t, err)

		// Assert
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func signRequest(t *testing.T, req *http.Request, creds *credentials.Credentials, region string) {
	t.Helper()

	signer := sigv4.NewSigner(creds)
	now := time.Now().Add(time.Duration(30) * time.Second)
	signedHeaders, err := signer.Sign(req, nil, "execute-api",
		region, now)
	assert.NoError(t, err)
	assert.NotNil(t, signedHeaders)
}