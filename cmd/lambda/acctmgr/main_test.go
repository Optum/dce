package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/Azure/go-autorest/autorest"
	"github.com/aws/aws-lambda-go/events"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	azureauthmock "github.com/Optum/Redbox/pkg/authorization/mocks"
	"github.com/Optum/Redbox/pkg/common"
	commock "github.com/Optum/Redbox/pkg/common/mocks"
	"github.com/Optum/Redbox/pkg/db"
	dbmock "github.com/Optum/Redbox/pkg/db/mocks"
	provmock "github.com/Optum/Redbox/pkg/provision/mocks"
)

type authorizationMock struct {
	mock.Mock
}

type databaseMock struct {
	mock.Mock
}

type jwtMock struct {
	mock.Mock
}

func (m *authorizationMock) ADGroupMember(ctx context.Context, groupID *string, memberID *string, tenantID *string) (result bool, err error) {
	if groupID != nil && memberID != nil {
		return true, nil
	}
	return false, errors.New("input Parameter Error")
}

func (m *authorizationMock) AddADGroupUser(ctx context.Context, memberID string, groupID string, tenantID string) (result autorest.Response, err error) {
	// Ridiculous mocking going on here, just pretend everything is fine.
	response := autorest.Response{
		Response: &http.Response{},
	}
	return response, nil
}

func (m *authorizationMock) RemoveADGroupUser(ctx context.Context, groupID string, memberID string, tenantID string) (result autorest.Response, err error) {
	// Ridiculous mocking going on here, just pretend everything is fine.
	response := autorest.Response{
		Response: &http.Response{},
	}
	return response, nil
}

func (m *databaseMock) UpdateAccount(id string, status string, userID string) error {
	if id != "" && status != "" && userID != "" {
		return nil
	}
	return errors.New("update failed")
}

func TestCheckGroupMembership(t *testing.T) {
	vals := common.ClaimKey{
		UserID:   "optum1",
		TenantID: "tenantid1",
		GroupID:  "7047a708-8c54-4570-9cfc-9a86e08935da",
	}

	authorization := new(authorizationMock)

	ctx := context.Background()
	testChkGrpMemResult, _ := authorization.ADGroupMember(ctx, &vals.GroupID, &vals.UserID, &vals.TenantID)

	expectedResult := true
	assert.Equal(t, expectedResult, testChkGrpMemResult)

	response := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(fmt.Sprintf("%v", testChkGrpMemResult)),
	}

	expectedResponse := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(fmt.Sprintf("%v", expectedResult)),
	}

	assert.Equal(t, expectedResponse, response)
}

func TestRouter(t *testing.T) {
	request := events.APIGatewayProxyRequest{
		HTTPMethod: "DELETE",
	}

	var response events.APIGatewayProxyResponse

	jwtMock := &commock.JWTTokenService{}

	jwtMock.On("ParseJWT").Return(common.ClaimKey{
		UserID:   "testuser1",
		GroupID:  "testgroup1",
		TenantID: "testtenant1",
	})

	if !(request.HTTPMethod == "GET") || !(request.HTTPMethod == "POST") || !(request.HTTPMethod == "PUT") {
		response = events.APIGatewayProxyResponse{
			StatusCode: http.StatusMethodNotAllowed,
			Body:       string("Method get/post/put are only allowed"),
		}
	} else {
		response = events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Body:       string("Method get/post/put are only allowed"),
		}
	}

	expectedResponse := events.APIGatewayProxyResponse{
		StatusCode: http.StatusMethodNotAllowed,
		Body:       string("Method get/post/put are only allowed"),
	}

	assert.Equal(t, expectedResponse, response)
}

// testProvisionAccountInput is the structure input used for table driven
// testing for provisionAccount
type testProvisionAccountInput struct {
	ExpectedResponse                        events.APIGatewayProxyResponse
	ExpectedError                           error
	GetReadyAccountAccount                  *db.RedboxAccount
	GetReadyAccountError                    error
	FindUserActiveAssignmentAssignment      *db.RedboxAccountAssignment
	FindUserActiveAssignmentError           error
	FindUserAssignmentWithAccountAssignment *db.RedboxAccountAssignment
	FindUserAssignmentWithAccountError      error
	ActivateAccountAssignmentError          error
	TransitionAccountStatusError            error
	AddADGroupUserResponse                  autorest.Response
	AddADGroupUserError                     error
	RollbackProvisionAccountError           error
}

// testProvisionAccount tests and verifies the flow of the function to
// provision an account for a user
func TestProvisionAccount(t *testing.T) {
	// Construct test scenarios
	tests := []testProvisionAccountInput{
		// Happy Path - Existing Assignment
		{
			ExpectedResponse: createResponse(201, "User successfully added to group "+
				"and Redbox account manifest has been updated. Your AWS account is "+
				"123. To login, please go to myapps.microsoft.com."),
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
			AddADGroupUserResponse: autorest.Response{
				Response: &http.Response{
					StatusCode: 204,
				},
			},
		},
		// Happy Path - New Assignment
		{
			ExpectedResponse: createResponse(201, "User successfully added to group "+
				"and Redbox account manifest has been updated. Your AWS account is "+
				"123. To login, please go to myapps.microsoft.com."),
			GetReadyAccountAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindUserActiveAssignmentAssignment:      &db.RedboxAccountAssignment{},
			FindUserAssignmentWithAccountAssignment: &db.RedboxAccountAssignment{},
			AddADGroupUserResponse: autorest.Response{
				Response: &http.Response{
					StatusCode: 204,
				},
			},
		},
		// Error Checking Assignments
		{
			ExpectedResponse: createResponse(503, "Cannot verify if User has "+
				"existing Redbox Account : Find Assignment Error"),
			ExpectedError:                 errors.New("Find Assignment Error"),
			FindUserActiveAssignmentError: errors.New("Find Assignment Error"),
		},
		// User already has an active account
		{
			ExpectedResponse: createResponse(409, "User already has an existing Redbox: 456"),
			ExpectedError:    errors.New("User already has an existing Redbox: 456"),
			FindUserActiveAssignmentAssignment: &db.RedboxAccountAssignment{
				UserID:           "abc",
				AccountID:        "456",
				AssignmentStatus: db.Active,
			},
		},
		// Error Getting Ready Accounts
		{
			ExpectedResponse: createResponse(503, "Cannot get Available Redbox "+
				"Accounts : Get Ready Account Error"),
			ExpectedError:                      errors.New("Get Ready Account Error"),
			FindUserActiveAssignmentAssignment: &db.RedboxAccountAssignment{},
			GetReadyAccountError:               errors.New("Get Ready Account Error"),
		},
		// No ready accounts
		{
			ExpectedResponse:                   createResponse(503, "No Available Redbox Accounts at this moment"),
			ExpectedError:                      errors.New("No Available Redbox Accounts at this moment"),
			FindUserActiveAssignmentAssignment: &db.RedboxAccountAssignment{},
			GetReadyAccountAccount:             nil,
		},
		// Error Finding User Assignment With Account
		{
			ExpectedResponse: createResponse(503, "Cannot get Available Redbox "+
				"Accounts : Find User Assignment with Account Error"),
			ExpectedError: errors.New("Find User Assignment with Account Error"),
			GetReadyAccountAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindUserActiveAssignmentAssignment: &db.RedboxAccountAssignment{},
			FindUserAssignmentWithAccountError: errors.New("Find User Assignment with Account Error"),
		},
		// Error Activate Account Assignment
		{
			ExpectedResponse: createResponse(500, "Failed to Create "+
				"Assignment for Account : 123"),
			ExpectedError: errors.New("Activate Account Assignment Error"),
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
			ExpectedResponse: createResponse(500, "Failed to Create "+
				"Assignment for Account : 123"),
			ExpectedError: errors.New("Transition Account Status Error"),
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
			ExpectedResponse: createResponse(500, "Failed to Rollback "+
				"Account Assignment for Account : 123"),
			ExpectedError: errors.New("Rollback Provision Account Error"),
			GetReadyAccountAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindUserActiveAssignmentAssignment:      &db.RedboxAccountAssignment{},
			FindUserAssignmentWithAccountAssignment: &db.RedboxAccountAssignment{},
			TransitionAccountStatusError:            errors.New("Transition Account Status Error"),
			RollbackProvisionAccountError:           errors.New("Rollback Provision Account Error"),
		},
		// Error Add AD Group User
		{
			ExpectedResponse: createResponse(500, "Fail to Add User abc for Account : 123"),
			ExpectedError:    errors.New("Add AD Group User Error"),
			GetReadyAccountAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindUserActiveAssignmentAssignment:      &db.RedboxAccountAssignment{},
			FindUserAssignmentWithAccountAssignment: &db.RedboxAccountAssignment{},
			AddADGroupUserError:                     errors.New("Add AD Group User Error"),
		},
		// Add AD Group User Non 204 Return
		{
			ExpectedResponse: createResponse(500, "Fail to Add User abc for Account : 123"),
			GetReadyAccountAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindUserActiveAssignmentAssignment:      &db.RedboxAccountAssignment{},
			FindUserAssignmentWithAccountAssignment: &db.RedboxAccountAssignment{},
			AddADGroupUserResponse: autorest.Response{
				Response: &http.Response{
					StatusCode: 500,
				},
			},
		},
		// Error Add AD Group User Rollback
		{
			ExpectedResponse: createResponse(500, "Failed to Rollback "+
				"Account Assignment for Account : 123"),
			ExpectedError: errors.New("Rollback Provision Account Error"),
			GetReadyAccountAccount: &db.RedboxAccount{
				ID:            "123",
				AccountStatus: db.Ready,
			},
			FindUserActiveAssignmentAssignment:      &db.RedboxAccountAssignment{},
			FindUserAssignmentWithAccountAssignment: &db.RedboxAccountAssignment{},
			AddADGroupUserError:                     errors.New("Add AD Group User Error"),
			RollbackProvisionAccountError:           errors.New("Rollback Provision Account Error"),
		},
	}

	// Iterate through each test in the list
	claimKey := common.ClaimKey{
		UserID:   "abc",
		TenantID: "def",
	}
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
			test.ActivateAccountAssignmentError)
		mockProv.On("RollbackProvisionAccount", mock.Anything, mock.Anything,
			mock.Anything).Return(test.RollbackProvisionAccountError)

		mockAuthor := &azureauthmock.Authorizationer{}
		mockAuthor.On("AddADGroupUser", mock.Anything, mock.Anything,
			mock.Anything, mock.Anything).Return(test.AddADGroupUserResponse,
			test.AddADGroupUserError)

		// Call provisionAccount
		response, err := provisionAccount(claimKey, mockDB, mockProv,
			mockAuthor)

		// Assert that the expected output is correct
		require.Equal(t, test.ExpectedResponse, response)
		require.Equal(t, test.ExpectedError, err)
	}
}

// testDecommissionAccountInput is the structure input used for table driven
// testing for decommissionAccount
type testDecommissionAccountInput struct {
	ExpectedResponse                events.APIGatewayProxyResponse
	ExpectedError                   error
	FindAssignmentByUserAssignments []*db.RedboxAccountAssignment
	FindAssignmentByUserError       error
	TransitionAssignmentStatusError error
	TransitionAccountStatusAccount  *db.RedboxAccount
	TransitionAccountStatusError    error
	RemoveADGroupUserResponse       autorest.Response
	RemoveADGroupUserError          error
	SendMessageError                error
}

// testDecommissionAccount tests and verifies the flow of the function to
// decommission an account for a user
func TestDecommissionAccount(t *testing.T) {
	// Construct test scenarios
	tests := []testDecommissionAccountInput{
		// Happy Path
		{
			ExpectedResponse: createResponse(200, "AWS Redbox Decommission: User 'abc' has been removed from the account group 'ghi'."),
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Active,
				},
			},
			TransitionAccountStatusAccount: &db.RedboxAccount{
				ID:            "123",
				GroupID:       "ghi",
				AccountStatus: db.Ready,
			},
			RemoveADGroupUserResponse: autorest.Response{
				Response: &http.Response{
					StatusCode: 204,
				},
			},
		},
		// Fail to find Assignment
		{
			ExpectedResponse:          createResponse(503, "Cannot verify if User has existing Redbox Account : Fail finding User Assignment"),
			ExpectedError:             errors.New("Fail finding User Assignment"),
			FindAssignmentByUserError: errors.New("Fail finding User Assignment"),
		},
		// Fail transition Assignment Status
		{
			ExpectedResponse: createResponse(500, "Failed Decommission on Account Assignment"),
			ExpectedError:    errors.New("Fail Assignment Status"),
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Active,
				},
			},
			TransitionAssignmentStatusError: errors.New("Fail Assignment Status"),
		},
		// Fail transition Account Status
		{
			ExpectedResponse: createResponse(500, "Failed Decommission on Account"),
			ExpectedError:    errors.New("Fail Account Status"),
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Active,
				},
			},
			TransitionAccountStatusError: errors.New("Fail Account Status"),
		},
		// Fail remove AD User from Group
		{
			ExpectedResponse: createResponse(500, "User has not been removed from group 'ghi'."),
			ExpectedError:    errors.New("Fail Remove AD User"),
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Active,
				},
			},
			TransitionAccountStatusAccount: &db.RedboxAccount{
				ID:            "123",
				GroupID:       "ghi",
				AccountStatus: db.Ready,
			},
			RemoveADGroupUserError: errors.New("Fail Remove AD User"),
		},
		// Fail sending Reset Message
		{
			ExpectedResponse: createResponse(500, "Failed to add Account 123 to be Reset."),
			ExpectedError:    errors.New("Fail Sending Message"),
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Active,
				},
			},
			TransitionAccountStatusAccount: &db.RedboxAccount{
				ID:            "123",
				GroupID:       "ghi",
				AccountStatus: db.Ready,
			},
			RemoveADGroupUserResponse: autorest.Response{
				Response: &http.Response{
					StatusCode: 204,
				},
			},
			SendMessageError: errors.New("Fail Sending Message"),
		},
	}

	// Iterate through each test in the list
	claimKey := common.ClaimKey{
		UserID:   "abc",
		TenantID: "def",
		GroupID:  "ghi",
	}
	queueURL := "url"
	for _, test := range tests {
		// Setup mocks
		mockDB := &dbmock.DBer{}
		mockDB.On("FindAssignmentByUser", mock.Anything).Return(
			test.FindAssignmentByUserAssignments,
			test.FindAssignmentByUserError)
		mockDB.On("TransitionAssignmentStatus", mock.Anything, mock.Anything,
			mock.Anything, mock.Anything).Return(nil,
			test.TransitionAssignmentStatusError)
		mockDB.On("TransitionAccountStatus", mock.Anything, mock.Anything,
			mock.Anything).Return(test.TransitionAccountStatusAccount,
			test.TransitionAccountStatusError)

		mockAuthor := &azureauthmock.Authorizationer{}
		mockAuthor.On("RemoveADGroupUser", mock.Anything, mock.Anything,
			mock.Anything, mock.Anything).Return(test.RemoveADGroupUserResponse,
			test.RemoveADGroupUserError)

		mockQueue := commock.Queue{}
		mockQueue.On("SendMessage", mock.Anything, mock.Anything).Return(
			test.SendMessageError)

		// Call decommissionAccount
		response, err := decommissionAccount(&claimKey, &queueURL, mockDB,
			&mockQueue, mockAuthor)

		// Assert that the expected output is correct
		require.Equal(t, test.ExpectedResponse, response)
		require.Equal(t, test.ExpectedError, err)
	}
}
