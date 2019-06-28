package reset

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Optum/Redbox/pkg/common"
)

// Launchpader interface for triggering a launchpad build on a provided
// account and check the current status of a launchpad build
type Launchpader interface {
	Setup(string) error                                      // Setup Launchpad to be Applied
	TriggerLaunchpad(string, string, string) (string, error) // Triggers Launchpad on an account under a master
	CheckLaunchpad(string, string, string) (string, error)   // Checks the status of an account's deployment
	Authenticate() (string, error)                           // Authenticate against the service
}

// LaunchpadAPI implements the Launchpader
type LaunchpadAPI struct {
	LaunchpadBaseEndpoint string              // The endpoint to hit when triggering Launchpad
	LaunchpadAuthEndpoint string              // The endpoint to get OAUTH token to trigger Launchpad
	ClientID              string              // The Client ID to authenticate to API
	ClientSecret          string              // The Client Secret to authenticate to API
	BackendBucket         string              // The S3 Bucket holding the Launchpad's tfstate
	HTTP                  common.HTTPClienter // The HTTP Client that will make the requests
	Storage               common.Storager     // Storage service to update the state file
	Token                 common.TokenService // Token service to assume role into an account
}

// LaunchpadStateFile is the structured json to replace the Terraform State file
// for Launchpad
type LaunchpadStateFile struct {
	Version          int    `json:"version"`
	TerraformVersion string `json:"terraform_version"`
	Serial           int    `json:"serial"`
	Lineage          string `json:"lineage"`
}

// Setup implementation to reset any existing Terraform Statefiles created by
// Launchpad. This applies an empty state file with just a version assignment
func (lp LaunchpadAPI) Setup(accountID string) error {
	// Get the Current State File
	key := fmt.Sprintf("bootstrap-launchpad-%s/terraform.state", accountID)
	log.Printf("Get Current State File: %s/%s\n", lp.BackendBucket, key)
	state, err := lp.Storage.GetObject(lp.BackendBucket, key)
	if err != nil {
		return err
	}

	// Create the new clean state file by truncating the module
	log.Println("Creating Clean State File")
	cleanState := LaunchpadStateFile{}
	err = json.Unmarshal([]byte(state), &cleanState)
	if err != nil {
		return err
	}

	// Verify the clean body has all the necessary fields
	if cleanState.Version == 0 {
		return fmt.Errorf("Error: No 'version' was found in clean state: %+v",
			cleanState)
	}
	if cleanState.TerraformVersion == "" {
		return fmt.Errorf("Error: No 'terraform_version' was found in clean state: %+v",
			cleanState)
	}
	if cleanState.Serial == 0 {
		return fmt.Errorf("Error: No 'serial' was found in clean state: %+v",
			cleanState)
	}
	if cleanState.Lineage == "" {
		return fmt.Errorf("Error: No 'lineage' was found in clean state: %+v",
			cleanState)
	}

	// Save the clean statefile temporarily
	cleanBody, err := json.Marshal(cleanState)
	if err != nil {
		return err
	}
	stateFile := "/tmp/temp-statefile"
	err = ioutil.WriteFile(stateFile, cleanBody, 0666)
	if err != nil {
		return err
	}
	defer os.Remove(stateFile)

	// Upload the clean state file
	log.Printf("Applying Clean State File: %s/%s\n", lp.BackendBucket, key)
	err = lp.Storage.Upload(lp.BackendBucket, key, stateFile)
	if err != nil {
		return err
	}
	log.Printf("Applying Clean State File Complete: %s/%s\n", lp.BackendBucket,
		key)

	return nil
}

// launchpadToken is the structured response from the Authenticate endpoint
// {
//   "token_type": "Bearer"
//   "expires_in": 3600,
//   "ext_expires_in": 3600,
//   "access_token": "eyJ0eXAiOiJKV1QiLCJhb..."
// }
type launchpadToken struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	ExtExpiresIn int    `json:"ext_expires_in"`
	AccessToken  string `json:"access_token"`
}

// Authenticate will authenticate the caller to be able to reach the launchpad
// api endpoint.
// Secrets should be stored in Parameter Store
func (lp LaunchpadAPI) Authenticate() (string, error) {
	// Create the request for the status api
	urlReq := lp.LaunchpadAuthEndpoint
	body := url.Values{}
	body.Set("grant_type", "client_credentials")
	body.Set("scope", "https://cloud.optum.com/.default")
	body.Set("client_id", lp.ClientID)
	body.Set("client_secret", lp.ClientSecret)
	request, err := http.NewRequest("POST", urlReq,
		strings.NewReader(body.Encode()))
	if err != nil {
		return "", err
	}

	// Set header and body
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Make and verify the response, then populate the auth
	auth := launchpadToken{}
	err = makeAndVerifyRequest(lp.HTTP, request, 200, &auth)
	if err != nil {
		return "", err
	}

	// Verify the access_token was actually retrieved
	if auth.AccessToken == "" {
		return "", errors.New("Error: No 'access_token' was found in response")
	}
	return auth.AccessToken, nil
}

// launchpadTrigger is the structured response from the Trigger endpoint
// {
//   "deploymentId": "string"
//   "deploymentStatusUrl": "string"
// }
type launchpadTrigger struct {
	DeploymentID        string `json:"deploymentId"`
	DeploymentStatusURL string `json:"deploymentStatusUrl"`
}

// launchpadTriggerPayload is the structure payload body for the Trigger
// endpoint
type launchpadTriggerPayload struct {
	MasterAccountName string `json:"masterAccountName"`
}

// TriggerLaunchpad implementation to hit API endpoint to trigger a launchpad
// build for the provided account
func (lp LaunchpadAPI) TriggerLaunchpad(accountID string, masterAccount string,
	bearerToken string) (string, error) {
	// Create the request for the trigger api
	url := fmt.Sprintf("%s/accounts/%s/deploys", lp.LaunchpadBaseEndpoint,
		accountID)
	payload := launchpadTriggerPayload{
		MasterAccountName: masterAccount,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	// Add a valid authenticator for the api
	bearer := fmt.Sprintf("Bearer %s", bearerToken)
	request.Header.Set("Authorization", bearer)
	request.Header.Set("Content-Type", "application/json")

	// Make and verify the response, then populate the trigger
	trigger := launchpadTrigger{}
	err = makeAndVerifyRequest(lp.HTTP, request, 201, &trigger)
	if err != nil {
		return "", err
	}

	// Verify the deploymentId was actually retrieved
	if trigger.DeploymentID == "" {
		return "", errors.New("Error: No 'deploymentId' was found in response")
	}
	return trigger.DeploymentID, nil
}

// launchpadCheck is the structured response from the Deploy State endpoint
// {
//   "status": "string"
// }
type launchpadCheck struct {
	Status string `json:"status"`
}

// CheckLaunchpad implementation to hit API endpoint to get the launchpad
// deployment status for the provided account and deployment id
//
// Status Returns:
//   "IN-PROGRESS"
//   "SUCCESS"
//   "ABORTED"
//   "FAILURE"
//   "UNSTABLE"
//   "NOT_BUILT"
// Based on pulling from Jenkins Builds
// https://github.com/jenkinsci/jenkins/blob/master/core/src/main/java/hudson/model/Result.java#L57
func (lp LaunchpadAPI) CheckLaunchpad(accountID string, deployID string,
	bearerToken string) (string, error) {
	// Create the request for the status api
	url := fmt.Sprintf("%s/accounts/%s/deploys/%s", lp.LaunchpadBaseEndpoint,
		accountID, deployID)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Add a valid authenticator for the api
	bearer := fmt.Sprintf("Bearer %s", bearerToken)
	request.Header.Set("Authorization", bearer)

	// Make and verify the response, then populate the check
	check := launchpadCheck{}
	err = makeAndVerifyRequest(lp.HTTP, request, 200, &check)
	if err != nil {
		return "", err
	}

	// Verify the status was actually retrieved
	if check.Status == "" {
		return "", errors.New("Error: No 'status' was found in response")
	}
	return check.Status, nil
}

// makeAndVerifyRequest is a helper function to make the http call with the
// provided request, verify the response, and populate the response struct
func makeAndVerifyRequest(httpClient common.HTTPClienter, req *http.Request,
	status int, respStruct interface{}) (rerr error) {
	// Make the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	// Verify the status code
	if resp.StatusCode != status {
		return fmt.Errorf("Returned a non %d response: %d", status,
			resp.StatusCode)
	}

	// Close the body after the reading from it
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			rerr = err
		}
	}()

	// Decode the body into the provided interface
	err = json.NewDecoder(resp.Body).Decode(respStruct)
	if err != nil {
		return err
	}

	return nil
}
