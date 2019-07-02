package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"github.com/Optum/Redbox/pkg/api/response"
	commock "github.com/Optum/Redbox/pkg/common/mocks"
	"github.com/Optum/Redbox/pkg/db"
	dbmock "github.com/Optum/Redbox/pkg/db/mocks"
	provmock "github.com/Optum/Redbox/pkg/provision/mocks"
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

// testPublishAssignmentInput is the structure input used for table driven
// testing for publishMessage
type testPublishAssignmentInput struct {
	ExpectedMessage     *string
	ExpectedError       error
	PublishMessageError error
}

// TestPublishAssignment tests and verifies the flow of the helper function
// publishAssignment to create and publish a message to an SNS Topic
func TestPublishAssignment(t *testing.T) {
	accountAssignment := &db.RedboxAccountAssignment{
		UserID:           "abc",
		AccountID:        "123",
		AssignmentStatus: db.Active,
		CreatedOn:        567,
		LastModifiedOn:   567,
	}
	accountAssignmentResponse :=
		response.CreateAccountAssignmentResponse(accountAssignment)
	accountAssignmentBytes, err :=
		json.Marshal(accountAssignmentResponse)
	if err != nil {
		log.Fatalf("Failed to Marshal Account Assignment: %s", err)
	}
	message := string(accountAssignmentBytes)

	// Construct test scenarios
	tests := []testPublishAssignmentInput{
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

		// Call publishAssignment
		message, err := publishAssignment(mockNotif, accountAssignment, &topic)

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
	ExpectedResponse                        events.APIGatewayProxyResponse
	GetReadyAccountAccount                  *db.RedboxAccount
	GetReadyAccountError                    error
	FindUserActiveAssignmentAssignment      *db.RedboxAccountAssignment
	FindUserActiveAssignmentError           error
	FindUserAssignmentWithAccountAssignment *db.RedboxAccountAssignment
	FindUserAssignmentWithAccountError      error
	ActivateAccountAssignmentAssignment     *db.RedboxAccountAssignment
	ActivateAccountAssignmentError          error
	TransitionAccountStatusError            error
	PublishMessageMessageID                 string
	PublishMessageError                     error
	RollbackProvisionAccountError           error
}

// TestProvisionAccount tests and verifies the flow of the function to
// provision an account for a user
func TestProvisionAccount(t *testing.T) {
	successfulAccountAssignment := &db.RedboxAccountAssignment{
		UserID:           "abc",
		AccountID:        "123",
		AssignmentStatus: db.Active,
		CreatedOn:        567,
		LastModifiedOn:   567,
	}
	successfulAccountAssignmentResponse :=
		response.CreateAccountAssignmentResponse(successfulAccountAssignment)
	successfulAccountAssignmentBytes, err :=
		json.Marshal(successfulAccountAssignmentResponse)
	if err != nil {
		log.Fatalf("Failed to Marshal Account Assignment: %s", err)
	}

	// Construct test scenarios
	tests := []testProvisionAccountInput{
		// Happy Path - Existing Assignment
		{
			ExpectedResponse: createAPIResponse(http.StatusCreated,
				string(successfulAccountAssignmentBytes)),
			GetReadyAccountAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindUserActiveAssignmentAssignment: &db.RedboxAccountAssignment{},
			FindUserAssignmentWithAccountAssignment: &db.RedboxAccountAssignment{
				UserID:           "abc",
				AccountID:        "123",
				AssignmentStatus: db.Decommissioned,
			},
			ActivateAccountAssignmentAssignment: successfulAccountAssignment,
		},
		// Happy Path - New Assignment
		{
			ExpectedResponse: createAPIResponse(http.StatusCreated,
				string(successfulAccountAssignmentBytes)),
			GetReadyAccountAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindUserActiveAssignmentAssignment:      &db.RedboxAccountAssignment{},
			FindUserAssignmentWithAccountAssignment: &db.RedboxAccountAssignment{},
			ActivateAccountAssignmentAssignment:     successfulAccountAssignment,
		},
		// Error Checking Assignments
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Cannot verify if User has existing Redbox Account : Find Assignment Error")),
			FindUserActiveAssignmentError: errors.New("Find Assignment Error"),
		},
		// User already has an active account
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusConflict,
				response.CreateErrorResponse("ClientError",
					"User already has an existing Redbox: 456")),
			FindUserActiveAssignmentAssignment: &db.RedboxAccountAssignment{
				UserID:           "abc",
				AccountID:        "456",
				AssignmentStatus: db.Active,
			},
		},
		// Error Getting Ready Accounts
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Cannot get Available Redbox Accounts : Get Ready Account Error")),
			FindUserActiveAssignmentAssignment: &db.RedboxAccountAssignment{},
			GetReadyAccountError:               errors.New("Get Ready Account Error"),
		},
		// No ready accounts
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusServiceUnavailable,
				response.CreateErrorResponse("ServerError",
					"No Available Redbox Accounts at this moment")),
			FindUserActiveAssignmentAssignment: &db.RedboxAccountAssignment{},
			GetReadyAccountAccount:             nil,
		},
		// Error Finding User Assignment With Account
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Cannot get Available Redbox Accounts : Find User Assignment with Account Error")),
			GetReadyAccountAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindUserActiveAssignmentAssignment: &db.RedboxAccountAssignment{},
			FindUserAssignmentWithAccountError: errors.New("Find User Assignment with Account Error"),
		},
		// Error Activate Account Assignment
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed to Create Assignment for Account : 123")),
			GetReadyAccountAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindUserActiveAssignmentAssignment:      &db.RedboxAccountAssignment{},
			FindUserAssignmentWithAccountAssignment: &db.RedboxAccountAssignment{},
			ActivateAccountAssignmentError:          errors.New("Activate Account Assignment Error"),
		},
		// Error Transition Account Status
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed to Create Assignment for 123 - abc")),
			GetReadyAccountAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindUserActiveAssignmentAssignment:      &db.RedboxAccountAssignment{},
			FindUserAssignmentWithAccountAssignment: &db.RedboxAccountAssignment{},
			TransitionAccountStatusError:            errors.New("Transition Account Status Error"),
		},
		// Error Transition Account Status Rollback
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed to Rollback Account Assignment for 123 - abc")),
			GetReadyAccountAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindUserActiveAssignmentAssignment:      &db.RedboxAccountAssignment{},
			FindUserAssignmentWithAccountAssignment: &db.RedboxAccountAssignment{},
			TransitionAccountStatusError:            errors.New("Transition Account Status Error"),
			RollbackProvisionAccountError:           errors.New("Rollback Provision Account Error"),
		},
		// Error Publish Message
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed to Create Assignment for 123 - abc")),
			GetReadyAccountAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindUserActiveAssignmentAssignment:      &db.RedboxAccountAssignment{},
			FindUserAssignmentWithAccountAssignment: &db.RedboxAccountAssignment{},
			ActivateAccountAssignmentAssignment:     &db.RedboxAccountAssignment{},
			PublishMessageError:                     errors.New("Publish Message Error"),
		},
		// Error Publish Message Rollback
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed to Rollback Account Assignment for 123 - abc")),
			GetReadyAccountAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindUserActiveAssignmentAssignment:      &db.RedboxAccountAssignment{},
			FindUserAssignmentWithAccountAssignment: &db.RedboxAccountAssignment{},
			ActivateAccountAssignmentAssignment:     &db.RedboxAccountAssignment{},
			PublishMessageError:                     errors.New("Publish Message Error"),
			RollbackProvisionAccountError:           errors.New("Rollback Provision Account Error"),
		},
	}

	// Iterate through each test in the list
	request := &requestBody{
		UserID: "abc",
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
		mockProv.On("FindUserActiveAssignment", mock.Anything).Return(
			test.FindUserActiveAssignmentAssignment,
			test.FindUserActiveAssignmentError)
		mockProv.On("FindUserAssignmentWithAccount", mock.Anything,
			mock.Anything).Return(
			test.FindUserAssignmentWithAccountAssignment,
			test.FindUserAssignmentWithAccountError)
		mockProv.On("ActivateAccountAssignment", mock.Anything,
			mock.Anything, mock.Anything).Return(
			test.ActivateAccountAssignmentAssignment,
			test.ActivateAccountAssignmentError)
		mockProv.On("RollbackProvisionAccount", mock.Anything, mock.Anything,
			mock.Anything).Return(test.RollbackProvisionAccountError)

		mockNotif := &commock.Notificationer{}
		mockNotif.On("PublishMessage", mock.Anything, mock.Anything,
			mock.Anything).Return(&test.PublishMessageMessageID,
			test.PublishMessageError)
		if test.FindUserActiveAssignmentError == nil &&
			test.GetReadyAccountError == nil &&
			test.GetReadyAccountAccount != nil &&
			test.FindUserAssignmentWithAccountError == nil &&
			test.TransitionAccountStatusError == nil &&
			test.ActivateAccountAssignmentError == nil &&
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
	ExpectedResponse                     events.APIGatewayProxyResponse
	FindAssignmentByUserAssignments      []*db.RedboxAccountAssignment
	FindAssignmentByUserError            error
	TransitionAssignmentStatusAssignment *db.RedboxAccountAssignment
	TransitionAssignmentStatusError      error
	TransitionAccountStatusAccount       *db.RedboxAccount
	TransitionAccountStatusError         error
	SendMessageError                     error
	PublishMessageMessageID              string
	PublishMessageError                  error
}

// TestDecommissionAccount tests and verifies the flow of the function to
// decommission an account for a user
func TestDecommissionAccount(t *testing.T) {
	successfulAccountAssignment := &db.RedboxAccountAssignment{
		UserID:           "abc",
		AccountID:        "123",
		AssignmentStatus: db.Decommissioned,
		CreatedOn:        567,
		LastModifiedOn:   567,
	}
	successfulAccountAssignmentResponse :=
		response.CreateAccountAssignmentResponse(successfulAccountAssignment)
	successfulAccountAssignmentBytes, err :=
		json.Marshal(successfulAccountAssignmentResponse)
	if err != nil {
		log.Fatalf("Failed to Marshal Account Assignment: %s", err)
	}

	// Construct test scenarios
	tests := []testDecommissionAccountInput{
		// Happy Path
		{
			ExpectedResponse: createAPIResponse(http.StatusOK,
				string(successfulAccountAssignmentBytes)),
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Active,
				},
			},
			TransitionAssignmentStatusAssignment: successfulAccountAssignment,
			TransitionAccountStatusAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
		},
		// Fail to find Assignment - No Assignments
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Cannot verify if User abc has a Redbox Assignment")),
			FindAssignmentByUserError: errors.New("Fail finding User Assignment"),
		},
		// Fail to find Assignment - No Active Assignments
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusBadRequest,
				response.CreateErrorResponse("ClientError",
					"No active account assignments found for abc")),
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "456",
					AssignmentStatus: db.Decommissioned,
				},
			},
		},
		// Fail to find Assignment - Assignment with Different ID
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusBadRequest,
				response.CreateErrorResponse("ClientError",
					"No active account assignments found for abc")),
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "456",
					AssignmentStatus: db.Active,
				},
			},
		},
		// Fail to decommission a Decommissioned Assignment
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusBadRequest,
				response.CreateErrorResponse("ClientError",
					"Account Assignment is not active for abc - 123")),
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Decommissioned,
				},
			},
		},
		// Fail transition Assignment Status
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed Decommission on Account Assignment abc - 123")),
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Active,
				},
			},
			TransitionAssignmentStatusError: errors.New("Fail Assignment Status"),
		},
		// Fail tranition Account Status
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed Decommission on Account Assignment abc - 123")),
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Active,
				},
			},
			TransitionAccountStatusError: errors.New("Fail Account Status"),
		},
		// Fail sending Reset Message
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed Decommission on Account Assignment abc - 123")),
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Active,
				},
			},
			TransitionAccountStatusAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			SendMessageError: errors.New("Fail Sending Message"),
		},
		// Error Publish Message
		{
			ExpectedResponse: createAPIErrorResponse(http.StatusInternalServerError,
				response.CreateErrorResponse("ServerError",
					"Failed Decommission on Account Assignment abc - 123")),
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Active,
				},
			},
			TransitionAccountStatusAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			TransitionAssignmentStatusAssignment: &db.RedboxAccountAssignment{},
			PublishMessageError:                  errors.New("Publish Message Error"),
		},
	}

	// Iterate through each test in the list
	request := &requestBody{
		UserID:    "abc",
		AccountID: "123",
	}
	queueURL := "url"
	topic := "topicARN"
	for _, test := range tests {
		// Setup mocks
		mockDB := &dbmock.DBer{}
		mockDB.On("FindAssignmentByUser", mock.Anything).Return(
			test.FindAssignmentByUserAssignments,
			test.FindAssignmentByUserError)
		mockDB.On("TransitionAssignmentStatus", mock.Anything, mock.Anything,
			mock.Anything, mock.Anything).Return(
			test.TransitionAssignmentStatusAssignment,
			test.TransitionAssignmentStatusError)
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
		if test.FindAssignmentByUserError == nil &&
			test.TransitionAssignmentStatusError == nil &&
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
