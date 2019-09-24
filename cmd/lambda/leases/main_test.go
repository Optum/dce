package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"github.com/Optum/Dcs/pkg/api/response"
	commock "github.com/Optum/Dcs/pkg/common/mocks"
	"github.com/Optum/Dcs/pkg/db"
	dbmock "github.com/Optum/Dcs/pkg/db/mocks"
	provmock "github.com/Optum/Dcs/pkg/provision/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testCreateAPIErrorResponseInput is the structure input used for table driven
// testing for createErrorResponse
type testCreateAPIErrorResponseInput struct {
	ExpectedResponse events.APIGatewayProxyResponse
	ResponseCode     int
	ErrResp          response.ErrorResponse
}

// TestCreateAPIErrorResponse tests and verifies the flow of the function to
// create a proper structure Error Response
func TestCreateAPIErrorResponse(t *testing.T) {
	// Construct test scenarios
	tests := []testCreateAPIErrorResponseInput{
		// Success 1
		{
			ExpectedResponse: events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: "{\"error\":{\"code\":\"ServerError\",\"message\":\"Server Side Error\"}}",
			},
			ResponseCode: http.StatusInternalServerError,
			ErrResp: response.CreateErrorResponse("ServerError",
				"Server Side Error"),
		},
		// Success 2
		{
			ExpectedResponse: events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: "{\"error\":{\"code\":\"ClientError\",\"message\":\"Client Side Error\"}}",
			},
			ResponseCode: http.StatusBadRequest,
			ErrResp: response.CreateErrorResponse("ClientError",
				"Client Side Error"),
		},
	}

	// Iterate through each test in the list
	for _, test := range tests {
		actualResponse := createAPIErrorResponse(test.ResponseCode, test.ErrResp)
		require.Equal(t, test.ExpectedResponse, actualResponse)
	}
}

// testPublishLeaseInput is the structure input used for table driven
// testing for publishMessage
type testPublishLeaseInput struct {
	ExpectedMessage     *string
	ExpectedError       error
	PublishMessageError error
}

// TestPublishLease tests and verifies the flow of the helper function
// publishLease to create and publish a message to an SNS Topic
func TestPublishLease(t *testing.T) {
	lease := &db.DcsLease{
		PrincipalID:           "abc",
		AccountID:             "123",
		LeaseStatus:           db.Active,
		CreatedOn:             567,
		LastModifiedOn:        567,
		LeaseStatusModifiedOn: 567,
	}
	leaseResponse :=
		response.CreateLeaseResponse(lease)
	leaseBytes, err :=
		json.Marshal(leaseResponse)
	if err != nil {
		log.Fatalf("Failed to Marshal Account Lease: %s", err)
	}
	message := string(leaseBytes)

	// Construct test scenarios
	tests := []testPublishLeaseInput{
		// Success
		{
			ExpectedMessage: &message,
		},
		// Failure
		{
			ExpectedError:       errors.New("Publish Message Error"),
			PublishMessageError: errors.New("Publish Message Error"),
		},
	}

	// Iterate through each test in the list
	messageID := "123"
	topic := "topicARN"
	for _, test := range tests {
		// Setup mocks
		mockNotif := &commock.Notificationer{}
		mockNotif.On("PublishMessage", mock.Anything, mock.Anything,
			mock.Anything).Return(&messageID, test.PublishMessageError)

		// Call publishLease
		message, err := publishLease(mockNotif, lease, &topic)

		// Assert that the expected message is correct
		if message == nil {
			require.Equal(t, test.ExpectedMessage, message)
		} else {
			require.Equal(t, *test.ExpectedMessage, *message)
		}
		require.Equal(t, test.ExpectedError, err)
	}
}

// testProvisionAccountInput is the structure input used for table driven
// testing for provisionAccount
type testProvisionAccountInput struct {
	ExpectedResponse                 events.APIGatewayProxyResponse
	GetReadyAccountAccount           *db.DcsAccount
	GetReadyAccountError             error
	FindActiveLeaseForPrincipal      *db.DcsLease
	FindActiveLeaseForPrincipalError error
	FindLeaseWithAccount             *db.DcsLease
	FindLeaseWithAccountError        error
	ActivateLease                    *db.DcsLease
	ActivateLeaseError               error
	TransitionAccountStatusError     error
	PublishMessageMessageID          string
	PublishMessageError              error
	RollbackProvisionAccountError    error
}

// TestProvisionAccount tests and verifies the flow of the function to
// provision an account for a Principal
func TestProvisionAccount(t *testing.T) {
	successfulLease := &db.DcsLease{
		PrincipalID:    "abc",
		AccountID:      "123",
		LeaseStatus:    db.Active,
		CreatedOn:      567,
		LastModifiedOn: 567,
	}
	successfulLeaseResponse :=
		response.CreateLeaseResponse(successfulLease)
	successfulLeaseBytes, err :=
		json.Marshal(successfulLeaseResponse)
	if err != nil {
		log.Fatalf("Failed to Marshal Account Lease: %s", err)
	}

	// Construct test scenarios
	tests := []testProvisionAccountInput{
		// Happy Path - Existing Lease
		{
			ExpectedResponse: createAPIResponse(http.StatusCreated,
				string(successfulLeaseBytes)),
			GetReadyAccountAccount: &db.DcsAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindActiveLeaseForPrincipal: &db.DcsLease{},
			FindLeaseWithAccount: &db.DcsLease{
				PrincipalID: "abc",
				AccountID:   "123",
				LeaseStatus: db.Decommissioned,
			},
			ActivateLease: successfulLease,
		},
		// Happy Path - New Lease
		{
			ExpectedResponse: createAPIResponse(http.StatusCreated,
				string(successfulLeaseBytes)),
			GetReadyAccountAccount: &db.DcsAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindActiveLeaseForPrincipal: &db.DcsLease{},
			FindLeaseWithAccount:        &db.DcsLease{},
			ActivateLease:               successfulLease,
		},
		// Error Checking Leases
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Cannot verify if Principal has existing Dcs Account : Find Lease Error")),
			FindActiveLeaseForPrincipalError: errors.New("Find Lease Error"),
		},
		// Principal already has an active account
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusConflict,
				response.CreateErrorResponse("ClientError",
					"Principal already has an existing Dcs: 456")),
			FindActiveLeaseForPrincipal: &db.DcsLease{
				PrincipalID: "abc",
				AccountID:   "456",
				LeaseStatus: db.Active,
			},
		},
		// Error Getting Ready Accounts
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Cannot get Available Dcs Accounts : Get Ready Account Error")),
			FindActiveLeaseForPrincipal: &db.DcsLease{},
			GetReadyAccountError:        errors.New("Get Ready Account Error"),
		},
		// No ready accounts
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusServiceUnavailable,
				response.CreateErrorResponse("ServerError",
					"No Available Dcs Accounts at this moment")),
			FindActiveLeaseForPrincipal: &db.DcsLease{},
			GetReadyAccountAccount:      nil,
		},
		// Error Finding Lease With Account
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Cannot get Available Dcs Accounts : Find Lease with Account Error")),
			GetReadyAccountAccount: &db.DcsAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindActiveLeaseForPrincipal: &db.DcsLease{},
			FindLeaseWithAccountError:   errors.New("Find Lease with Account Error"),
		},
		// Error Activate Account Lease
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed to Create Lease for Account : 123")),
			GetReadyAccountAccount: &db.DcsAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindActiveLeaseForPrincipal: &db.DcsLease{},
			FindLeaseWithAccount:        &db.DcsLease{},
			ActivateLeaseError:          errors.New("Activate Account Lease Error"),
		},
		// Error Transition Account Status
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed to Create Lease for 123 - abc")),
			GetReadyAccountAccount: &db.DcsAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindActiveLeaseForPrincipal:  &db.DcsLease{},
			FindLeaseWithAccount:         &db.DcsLease{},
			TransitionAccountStatusError: errors.New("Transition Account Status Error"),
		},
		// Error Transition Account Status Rollback
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed to Rollback Account Lease for 123 - abc")),
			GetReadyAccountAccount: &db.DcsAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindActiveLeaseForPrincipal:   &db.DcsLease{},
			FindLeaseWithAccount:          &db.DcsLease{},
			TransitionAccountStatusError:  errors.New("Transition Account Status Error"),
			RollbackProvisionAccountError: errors.New("Rollback Provision Account Error"),
		},
		// Error Publish Message
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed to Create Lease for 123 - abc")),
			GetReadyAccountAccount: &db.DcsAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindActiveLeaseForPrincipal: &db.DcsLease{},
			FindLeaseWithAccount:        &db.DcsLease{},
			ActivateLease:               &db.DcsLease{},
			PublishMessageError:         errors.New("Publish Message Error"),
		},
		// Error Publish Message Rollback
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed to Rollback Account Lease for 123 - abc")),
			GetReadyAccountAccount: &db.DcsAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindActiveLeaseForPrincipal:   &db.DcsLease{},
			FindLeaseWithAccount:          &db.DcsLease{},
			ActivateLease:                 &db.DcsLease{},
			PublishMessageError:           errors.New("Publish Message Error"),
			RollbackProvisionAccountError: errors.New("Rollback Provision Account Error"),
		},
	}

	// Iterate through each test in the list
	request := &requestBody{
		PrincipalID:              "abc",
		BudgetAmount:             350,
		BudgetCurrency:           "USD",
		BudgetNotificationEmails: []string{"test@test.com"},
	}
	topic := "topicARN"
	for _, test := range tests {
		// Setup mocks
		mockDB := &dbmock.DBer{}
		mockDB.On("GetReadyAccount").Return(test.GetReadyAccountAccount,
			test.GetReadyAccountError)
		mockDB.On("TransitionAccountStatus", mock.Anything, mock.Anything,
			mock.Anything).Return(nil, test.TransitionAccountStatusError)

		mockProv := &provmock.Provisioner{}
		mockProv.On("FindActiveLeaseForPrincipal", mock.Anything).Return(
			test.FindActiveLeaseForPrincipal,
			test.FindActiveLeaseForPrincipalError)
		mockProv.On("FindLeaseWithAccount", mock.Anything,
			mock.Anything).Return(
			test.FindLeaseWithAccount,
			test.FindLeaseWithAccountError)
		mockProv.On("ActivateAccount", mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			test.ActivateLease,
			test.ActivateLeaseError)
		mockProv.On("RollbackProvisionAccount", mock.Anything, mock.Anything,
			mock.Anything).Return(test.RollbackProvisionAccountError)

		mockNotif := &commock.Notificationer{}
		mockNotif.On("PublishMessage", mock.Anything, mock.Anything,
			mock.Anything).Return(&test.PublishMessageMessageID,
			test.PublishMessageError)
		if test.FindActiveLeaseForPrincipalError == nil &&
			test.GetReadyAccountError == nil &&
			test.GetReadyAccountAccount != nil &&
			test.FindLeaseWithAccountError == nil &&
			test.TransitionAccountStatusError == nil &&
			test.ActivateLeaseError == nil &&
			test.RollbackProvisionAccountError == nil {
			defer mockNotif.AssertExpectations(t)
		}

		// Call provisionAccount
		response := provisionAccount(request, mockDB, mockNotif,
			mockProv, &topic)

		// Assert that the expected output is correct
		require.Equal(t, test.ExpectedResponse, response)
	}
}

// testDecommissionAccountInput is the structure input used for table driven
// testing for decommissionAccount
type testDecommissionAccountInput struct {
	ExpectedResponse               events.APIGatewayProxyResponse
	FindLeaseByLeases              []*db.DcsLease
	FindLeaseByPrincipalError      error
	TransitionLeaseStatusLease     *db.DcsLease
	TransitionLeaseStatusError     error
	TransitionAccountStatusAccount *db.DcsAccount
	TransitionAccountStatusError   error
	SendMessageError               error
	PublishMessageMessageID        string
	PublishMessageError            error
}

// TestDecommissionAccount tests and verifies the flow of the function to
// decommission an account for a Principal
func TestDecommissionAccount(t *testing.T) {
	successfulLease := &db.DcsLease{
		PrincipalID:    "abc",
		AccountID:      "123",
		LeaseStatus:    db.Decommissioned,
		CreatedOn:      567,
		LastModifiedOn: 567,
	}
	successfulLeaseResponse :=
		response.CreateLeaseResponse(successfulLease)
	successfulLeaseBytes, err :=
		json.Marshal(successfulLeaseResponse)
	if err != nil {
		log.Fatalf("Failed to Marshal Account Lease: %s", err)
	}

	// Construct test scenarios
	tests := []testDecommissionAccountInput{
		// Happy Path
		{
			ExpectedResponse: createAPIResponse(http.StatusOK,
				string(successfulLeaseBytes)),
			FindLeaseByLeases: []*db.DcsLease{
				&db.DcsLease{
					PrincipalID: "abc",
					AccountID:   "123",
					LeaseStatus: db.Active,
				},
			},
			TransitionLeaseStatusLease: successfulLease,
			TransitionAccountStatusAccount: &db.DcsAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
		},
		// Fail to find Lease - No Leases
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Cannot verify if Principal abc has a Dcs Lease")),
			FindLeaseByPrincipalError: errors.New("Fail finding Lease"),
		},
		// Fail to find Lease - No Active Leases
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusBadRequest,
				response.CreateErrorResponse("ClientError",
					"No active account leases found for abc")),
			FindLeaseByLeases: []*db.DcsLease{
				&db.DcsLease{
					PrincipalID: "abc",
					AccountID:   "456",
					LeaseStatus: db.Decommissioned,
				},
			},
		},
		// Fail to find Lease - Lease with Different ID
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusBadRequest,
				response.CreateErrorResponse("ClientError",
					"No active account leases found for abc")),
			FindLeaseByLeases: []*db.DcsLease{
				&db.DcsLease{
					PrincipalID: "abc",
					AccountID:   "456",
					LeaseStatus: db.Active,
				},
			},
		},
		// Fail to decommission a Decommissioned Lease
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusBadRequest,
				response.CreateErrorResponse("ClientError",
					"Account Lease is not active for abc - 123")),
			FindLeaseByLeases: []*db.DcsLease{
				&db.DcsLease{
					PrincipalID: "abc",
					AccountID:   "123",
					LeaseStatus: db.Decommissioned,
				},
			},
		},
		// Fail transition Lease Status
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed Decommission on Account Lease abc - 123")),
			FindLeaseByLeases: []*db.DcsLease{
				&db.DcsLease{
					PrincipalID: "abc",
					AccountID:   "123",
					LeaseStatus: db.Active,
				},
			},
			TransitionLeaseStatusError: errors.New("Fail Lease Status"),
		},
		// Fail tranition Account Status
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed Decommission on Account Lease abc - 123")),
			FindLeaseByLeases: []*db.DcsLease{
				&db.DcsLease{
					PrincipalID: "abc",
					AccountID:   "123",
					LeaseStatus: db.Active,
				},
			},
			TransitionAccountStatusError: errors.New("Fail Account Status"),
		},
		// Fail sending Reset Message
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed Decommission on Account Lease abc - 123")),
			FindLeaseByLeases: []*db.DcsLease{
				&db.DcsLease{
					PrincipalID: "abc",
					AccountID:   "123",
					LeaseStatus: db.Active,
				},
			},
			TransitionAccountStatusAccount: &db.DcsAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			SendMessageError: errors.New("Fail Sending Message"),
		},
		// Error Publish Message
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed Decommission on Account Lease abc - 123")),
			FindLeaseByLeases: []*db.DcsLease{
				&db.DcsLease{
					PrincipalID: "abc",
					AccountID:   "123",
					LeaseStatus: db.Active,
				},
			},
			TransitionAccountStatusAccount: &db.DcsAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			TransitionLeaseStatusLease: &db.DcsLease{},
			PublishMessageError:        errors.New("Publish Message Error"),
		},
	}

	// Iterate through each test in the list
	request := &requestBody{
		PrincipalID: "abc",
		AccountID:   "123",
	}
	queueURL := "url"
	topic := "topicARN"
	for _, test := range tests {
		// Setup mocks
		mockDB := &dbmock.DBer{}
		mockDB.On("FindLeasesByPrincipal", mock.Anything).Return(
			test.FindLeaseByLeases,
			test.FindLeaseByPrincipalError)
		mockDB.On("TransitionLeaseStatus", mock.Anything, mock.Anything,
			mock.Anything, mock.Anything).Return(
			test.TransitionLeaseStatusLease,
			test.TransitionLeaseStatusError)
		mockDB.On("TransitionAccountStatus", mock.Anything, mock.Anything,
			mock.Anything).Return(test.TransitionAccountStatusAccount,
			test.TransitionAccountStatusError)

		mockQueue := commock.Queue{}
		mockQueue.On("SendMessage", mock.Anything, mock.Anything).Return(
			test.SendMessageError)

		mockNotif := &commock.Notificationer{}
		mockNotif.On("PublishMessage", mock.Anything, mock.Anything,
			mock.Anything).Return(&test.PublishMessageMessageID,
			test.PublishMessageError)
		if test.FindLeaseByPrincipalError == nil &&
			test.TransitionLeaseStatusError == nil &&
			test.TransitionAccountStatusError == nil &&
			test.SendMessageError == nil &&
			test.ExpectedResponse.StatusCode != 400 {
			defer mockNotif.AssertExpectations(t)
		}

		// Call decommissionAccount
		response := decommissionAccount(request, &queueURL, mockDB,
			&mockQueue, mockNotif, &topic)

		// Assert that the expected output is correct
		require.Equal(t, test.ExpectedResponse, response)
	}
}
