package reset

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockStorager is a mocked implementation of Storager
type mockStorager struct {
	mock.Mock
}

// GetObject is used for testing
func (mock mockStorager) GetObject(bucket string, key string) (string, error) {
	args := mock.Called(bucket, key)
	body := args.String(0)
	err := args.Error(1)
	return body, err
}

// PutObject is used for testing
func (mock mockStorager) PutObject(bucket string, key string,
	body io.ReadSeeker) error {
	args := mock.Called(bucket, key, body)
	err := args.Error(0)
	return err
}

// testLaunchpadSetupInput is the structure input used for table driven testing
// for LaunchpadAPI.Setup
type testLaunchpadSetupInput struct {
	Error           error
	GetObjectBody   string
	GetObjectError  error
	PutObjectError  error
	ExpectPutObject bool
}

// TestSetup verifies the flow of LaunchpadAPI.Setup is correct in that
// it can replace the necessary state file in the Storager
func TestSetup(t *testing.T) {
	// Construct test scenarios
	tests := []testLaunchpadSetupInput{
		// Happy Path Test
		{
			Error:           nil,
			GetObjectBody:   `{"version":3,"terraform_version":"0.11.7","serial":50,"lineage":"7db5af4f-7014-377c-388e-1a9bfd7d4413","extra":[]} `,
			GetObjectError:  nil,
			PutObjectError:  nil,
			ExpectPutObject: true,
		},
		// GetObject Failure
		{
			Error:           errors.New("Error : Failed to Get Object"),
			GetObjectBody:   `{"version":3,"terraform_version":"0.11.7","serial":50,"lineage":"7db5af4f-7014-377c-388e-1a9bfd7d4413"} `,
			GetObjectError:  errors.New("Error : Failed to Get Object"),
			PutObjectError:  nil,
			ExpectPutObject: false,
		},
		// Invalid State File Version
		{
			Error:           errors.New("Error: No 'version' was found in clean state: {Version:0 TerraformVersion: Serial:0 Lineage:}"),
			GetObjectBody:   `{} `,
			GetObjectError:  nil,
			PutObjectError:  nil,
			ExpectPutObject: false,
		},
		// Invalid State File Terraform Version
		{
			Error:           errors.New("Error: No 'terraform_version' was found in clean state: {Version:3 TerraformVersion: Serial:0 Lineage:}"),
			GetObjectBody:   `{"version":3}`,
			GetObjectError:  nil,
			PutObjectError:  nil,
			ExpectPutObject: false,
		},
		// Invalid State File Serial
		{
			Error:           errors.New("Error: No 'serial' was found in clean state: {Version:3 TerraformVersion:0.11.7 Serial:0 Lineage:}"),
			GetObjectBody:   `{"version":3,"terraform_version":"0.11.7"}`,
			GetObjectError:  nil,
			PutObjectError:  nil,
			ExpectPutObject: false,
		},
		// Invalid State File Lineage
		{
			Error:           errors.New("Error: No 'lineage' was found in clean state: {Version:3 TerraformVersion:0.11.7 Serial:50 Lineage:}"),
			GetObjectBody:   `{"version":3,"terraform_version":"0.11.7","serial":50}`,
			GetObjectError:  nil,
			PutObjectError:  nil,
			ExpectPutObject: false,
		},
		// PutObject Failure
		{
			Error:           errors.New("Error : Failed to Put Object"),
			GetObjectBody:   `{"version":3,"terraform_version":"0.11.7","serial":50,"lineage":"7db5af4f-7014-377c-388e-1a9bfd7d4413"} `,
			GetObjectError:  nil,
			PutObjectError:  errors.New("Error : Failed to Put Object"),
			ExpectPutObject: true,
		},
	}

	// Iterate through each test in the list
	account := "111111111111"
	backendBucket := "backend-bucket"
	stateKey := "bootstrap-launchpad-111111111111/terraform.state"
	for _, test := range tests {
		// Set up mocks
		storage := mockStorager{}
		storage.On("GetObject", backendBucket, stateKey).Return(
			test.GetObjectBody, test.GetObjectError)
		if test.ExpectPutObject {
			storage.On("PutObject", backendBucket, stateKey,
				mock.Anything).Return(test.PutObjectError)
		}

		// Create the LaunchpadAPI
		launchpad := LaunchpadAPI{
			BackendBucket: backendBucket,
			Storage:       storage,
		}

		// Call Setup
		err := launchpad.Setup(account)
		storage.AssertExpectations(t)

		// Assert that the expected output is correct
		require.Equal(t, test.Error, err)
	}
}

// mockHTTP is a mocked implementation of HTTPClienter
type mockHTTP struct {
	mock.Mock
}

// Do is used for testing
func (mock mockHTTP) Do(request *http.Request) (*http.Response, error) {
	args := mock.Called(request)
	statusCode := args.Int(0)
	body := ioutil.NopCloser(bytes.NewReader([]byte(args.String(1))))
	err := args.Error(2)
	response := http.Response{
		StatusCode: statusCode,
		Body:       body,
	}
	return &response, err
}

// testLaunchpadLaunchpadInput is the structure input used for table driven
// testing for LaunchpadAPI.TriggerLaunchpad
type testTriggerLaunchpadInput struct {
	DoStatus int
	DoBody   string
	DoError  error
	ID       string
	Error    error
}

// TestTriggerLaunchpad verifies the flow of LaunchpadAPI.TriggerLaunchpad is
// that it can resolve correclty based on the Launchpad Request
func TestTriggerLaunchpad(t *testing.T) {
	// Construct test scenarios
	tests := []testTriggerLaunchpadInput{
		// Happy Path Test
		{
			DoStatus: 201,
			DoBody:   `{"deploymentId":"123","deploymentStatusUrl":"deploys/123"}`,
			DoError:  nil,
			ID:       "123",
			Error:    nil,
		},
		// Error making call
		{
			DoStatus: 500,
			DoBody:   "",
			DoError:  errors.New("Error making request"),
			ID:       "",
			Error:    errors.New("Error making request"),
		},
		// Non 201 Request
		{
			DoStatus: 401,
			DoBody:   "",
			DoError:  nil,
			ID:       "",
			Error:    errors.New("Returned a non 201 response: 401"),
		},
		// No deploymentId
		{
			DoStatus: 201,
			DoBody:   `{"deploymentStatusUrl":"deploys/123"}`,
			DoError:  nil,
			ID:       "",
			Error:    errors.New("Error: No 'deploymentId' was found in response"),
		},
	}

	// Iterate through each test in the list
	launchpadBaseEndpoint := "http://mock.launchpad.com/api/v1"
	account := "111111111111"
	masterAccount := "TEST"
	bearer := "abcdefg"
	for _, test := range tests {
		// Set up mocks
		httpClient := mockHTTP{}
		httpClient.On("Do", mock.Anything).Return(test.DoStatus, test.DoBody,
			test.DoError)

		// Create the LaunchpadAPI
		launchpad := LaunchpadAPI{
			LaunchpadBaseEndpoint: launchpadBaseEndpoint,
			HTTP:                  httpClient,
		}

		// Call Setup
		id, err := launchpad.TriggerLaunchpad(account, masterAccount, bearer)

		// Assert that the expected output is correct
		require.Equal(t, test.ID, id)
		require.Equal(t, test.Error, err)
	}
}

// testLaunchpadCheckInput is the structure input used for table driven testing
// for LaunchpadAPI.CheckLaunchpad
type testCheckLaunchpadInput struct {
	DoStatus int
	DoBody   string
	DoError  error
	Status   string
	Error    error
}

// TestCheckLaunchpad verifies the flow of LaunchpadAPI.CheckLaunchpad is
// that it can resolve correctly based on the Launchpad Request
func TestCheckLaunchpad(t *testing.T) {
	// Construct test scenarios
	tests := []testCheckLaunchpadInput{
		// Happy Path SUCCESS
		{
			DoStatus: 200,
			DoBody:   `{"status":"SUCCESS"}`,
			DoError:  nil,
			Status:   "SUCCESS",
			Error:    nil,
		},
		// Happy Path any other status
		{
			DoStatus: 200,
			DoBody:   `{"status":"MY-STATUS"}`,
			DoError:  nil,
			Status:   "MY-STATUS",
			Error:    nil,
		},
		// Error making call
		{
			DoStatus: 500,
			DoBody:   "",
			DoError:  errors.New("Error making request"),
			Status:   "",
			Error:    errors.New("Error making request"),
		},
		// Non 200 Request
		{
			DoStatus: 401,
			DoBody:   "",
			DoError:  nil,
			Status:   "",
			Error:    errors.New("Returned a non 200 response: 401"),
		},
		// No status
		{
			DoStatus: 200,
			DoBody:   `{}`,
			DoError:  nil,
			Status:   "",
			Error:    errors.New("Error: No 'status' was found in response"),
		},
	}

	// Iterate through each test in the list
	launchpadBaseEndpoint := "http://mock.launchpad.com/api/v1"
	account := "111111111111"
	deployID := "123"
	bearer := "abcdefg"
	for _, test := range tests {
		// Set up mocks
		httpClient := mockHTTP{}
		httpClient.On("Do", mock.Anything).Return(test.DoStatus, test.DoBody,
			test.DoError)

		// Create the LaunchpadAPI
		launchpad := LaunchpadAPI{
			LaunchpadBaseEndpoint: launchpadBaseEndpoint,
			HTTP:                  httpClient,
		}

		// Call Setup
		status, err := launchpad.CheckLaunchpad(account, deployID, bearer)

		// Assert that the expected output is correct
		require.Equal(t, test.Status, status)
		require.Equal(t, test.Error, err)
	}
}

// testAuthenticateInput is the structure input used for table driven testing
// for LaunchpadAPI.Authenticate
type testAuthenticateInput struct {
	DoStatus int
	DoBody   string
	DoError  error
	Bearer   string
	Error    error
}

// TestAuthenticate verifies the flow of LaunchpadAPI.Authenticate is
// that it can resolve correctly based on the Launchpad OAUTH Request
func TestAuthenticate(t *testing.T) {
	// Construct test scenarios
	tests := []testAuthenticateInput{
		// Happy Path SUCCESS
		{
			DoStatus: 200,
			DoBody:   `{"token_type":"","expires_in":0,"ext_expires_in":0,"access_token":"123456"}`,
			DoError:  nil,
			Bearer:   "123456",
			Error:    nil,
		},
		// Error making call
		{
			DoStatus: 500,
			DoBody:   "",
			DoError:  errors.New("Error making request"),
			Bearer:   "",
			Error:    errors.New("Error making request"),
		},
		// Non 200 Request
		{
			DoStatus: 401,
			DoBody:   "",
			DoError:  nil,
			Bearer:   "",
			Error:    errors.New("Returned a non 200 response: 401"),
		},
		// No access_token
		{
			DoStatus: 200,
			DoBody:   `{"token_type":""}`,
			DoError:  nil,
			Bearer:   "",
			Error:    errors.New("Error: No 'access_token' was found in response"),
		},
	}

	// Iterate through each test in the list
	clientID := "abcdef"
	clientSecret := "ghijkl"
	for _, test := range tests {
		// Set up mocks
		httpClient := mockHTTP{}
		httpClient.On("Do", mock.Anything).Return(test.DoStatus, test.DoBody,
			test.DoError)

		// Create the LaunchpadAPI
		launchpad := LaunchpadAPI{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			HTTP:         httpClient,
		}

		// Call Setup
		bearer, err := launchpad.Authenticate()

		// Assert that the expected output is correct
		require.Equal(t, test.Bearer, bearer)
		require.Equal(t, test.Error, err)
	}
}

// testResponse is the response structure used to test the makeAndVerifyRequst
type testResponse struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// testMakeAndVerifyRequest is the structure input used for table driven testing
// for helper function makeAndVerifyRequest
type testMakeAndVerifyRequest struct {
	DoStatus int
	DoBody   string
	DoError  error
	Bearer   string
	Error    error
}

// TestMakeAndVerifyRequest verifies the helper function can make a request and
// process the response correctly and build out the correct structure
func TestMakeAndVerifyRequest(t *testing.T) {
	// Construct test scenarios
	tests := []testAuthenticateInput{
		// Happy Path SUCCESS
		{
			DoStatus: 200,
			DoBody:   `{"name":"ant","age":101}`,
			DoError:  nil,
			Bearer:   "123456",
			Error:    nil,
		},
		// Error making call
		{
			DoStatus: 500,
			DoBody:   "",
			DoError:  errors.New("Error making request"),
			Bearer:   "",
			Error:    errors.New("Error making request"),
		},
		// Incorrect Request Status
		{
			DoStatus: 401,
			DoBody:   "",
			DoError:  nil,
			Bearer:   "",
			Error:    errors.New("Returned a non 200 response: 401"),
		},
	}

	// Iterate through each test in the list
	request := http.Request{}
	response := testResponse{}
	status := 200
	for _, test := range tests {
		// Set up mocks
		httpClient := mockHTTP{}
		httpClient.On("Do", mock.Anything).Return(test.DoStatus, test.DoBody,
			test.DoError)

		// Call Setup
		err := makeAndVerifyRequest(httpClient, &request, status, &response)

		// Assert that the expected output is correct
		require.Equal(t, test.Error, err)
	}
}
