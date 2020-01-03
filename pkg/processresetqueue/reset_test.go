package processresetqueue

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/stretchr/testify/mock"

	comMocks "github.com/Optum/dce/pkg/common/mocks"
	"github.com/Optum/dce/pkg/db"
	dbMocks "github.com/Optum/dce/pkg/db/mocks"
	"github.com/stretchr/testify/require"

	"github.com/Optum/dce/pkg/common"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// mockQueue will implement and mock the Queue interface for testing the Reset
type mockQueue struct {
	NumLoop int
}

// createMockQueue constructs a mockQueue and returns its pointer
func createMockQueue(numLoop int) *mockQueue {
	return &mockQueue{NumLoop: numLoop}
}

// SendMessage mocks the interface function, not used
func (queue *mockQueue) SendMessage(queueURL *string, message *string) error {
	return nil
}

func (queue *mockQueue) NewFromEnv() error {
	return nil
}

// ReceiveMessage method returns an AWS SQS Message Output based on the provided
// Message Input. Used for testing.
func (queue *mockQueue) ReceiveMessage(input *sqs.ReceiveMessageInput) (
	*sqs.ReceiveMessageOutput, error) {
	// Fail if the queue url is the failure trigger
	if *input.QueueUrl == "https://mytesturl.com/123456789012/fail_receive" {
		return nil, errors.New("Error: Fail to Receive Message")
	}

	// Create the receiptHandle, this will be used to fail under DeleteMessage
	urlArray := strings.Split(*input.QueueUrl, "/")
	receiptHandle := urlArray[len(urlArray)-1]

	// Create a Message if numLoop is still positive
	messages := []*sqs.Message{}
	if queue.NumLoop > 0 {
		body := fmt.Sprintf("accountId-%d", queue.NumLoop)
		receiptHandle := receiptHandle
		messageID := "messageId"
		messages = append(messages, &sqs.Message{
			Body:          &body,
			ReceiptHandle: &receiptHandle,
			MessageId:     &messageID,
		})
	}
	queue.NumLoop--

	// Create ReceiveMessageOutput
	receiveMesssgeOutput := sqs.ReceiveMessageOutput{
		Messages: messages,
	}
	return &receiveMesssgeOutput, nil
}

// DeleteMessage method returns an AWS SQS Delete Message Output based on the
// provided Delete Message Input. Used for testing.
func (queue *mockQueue) DeleteMessage(input *sqs.DeleteMessageInput) (
	*sqs.DeleteMessageOutput, error) {
	// Fail if the ReceiptHandle is the failure trigger
	if *input.ReceiptHandle == "fail_delete" {
		return nil, errors.New("Error: Fail to Delete Message")
	}

	// Else return an empty DeleteMessageOutput
	return &sqs.DeleteMessageOutput{}, nil
}

// resetTest is the testing structure used for table driven testing on the
// Reset Function
type resetTest struct {
	ResetQueue         common.Queue
	ResetQueueURL      string
	BuildError         error
	ExpectedBuildEnv   map[string]string
	ExpectedBuildCount int
	GetAccount         *db.Account
	GetAccountError    error
	ExpectedOutput     *ResetOutput
	ExpectedError      error
}

// TestReset verifies the Reset function works as intended, where it should be
// able to digest from a Queue and trigger the respective Pipelines, while
// returning a correct ResetOutput response.
func TestReset(t *testing.T) {

	// Set up the list of tests to execute
	tests := []resetTest{
		// Test with No Messages Received
		{
			ResetQueue:    createMockQueue(0),
			ResetQueueURL: "https://mytesturl.com/123456789012/reset_queue",
			GetAccount: &db.Account{
				ID:               "1234567890",
				AdminRoleArn:     "arn:aws:iam::123456789012:role/AdminRole",
				PrincipalRoleArn: "arn:aws:iam::123456789012:role/PrincipalRole",
			},
			ExpectedBuildCount: 0,
			ExpectedBuildEnv: map[string]string{
				"RESET_ACCOUNT":                     "1234567890",
				"RESET_ACCOUNT_ADMIN_ROLE_NAME":     "AdminRole",
				"RESET_ACCOUNT_PRINCIPAL_ROLE_NAME": "PrincipalRole",
			},
			ExpectedOutput: &ResetOutput{
				Success:  true,
				Accounts: map[string]ResetResult{},
			},
			ExpectedError: nil,
		},
		// Test with 1 Message Received
		{
			ResetQueue:    createMockQueue(1),
			ResetQueueURL: "https://mytesturl.com/123456789012/reset_queue",
			GetAccount: &db.Account{
				ID:               "1234567890",
				AdminRoleArn:     "arn:aws:iam::123456789012:role/AdminRole",
				PrincipalRoleArn: "arn:aws:iam::123456789012:role/PrincipalRole",
			},
			ExpectedBuildCount: 1,
			ExpectedBuildEnv: map[string]string{
				"RESET_ACCOUNT":                     "1234567890",
				"RESET_ACCOUNT_ADMIN_ROLE_NAME":     "AdminRole",
				"RESET_ACCOUNT_PRINCIPAL_ROLE_NAME": "PrincipalRole",
			},
			ExpectedOutput: &ResetOutput{
				Success: true,
				Accounts: map[string]ResetResult{
					"accountId-1": ResetResult{
						BuildTrigger:    true,
						MessageDeletion: true,
					},
				},
			},
			ExpectedError: nil,
		},
		// Test with 5 Message Received
		{
			ResetQueue:    createMockQueue(5),
			ResetQueueURL: "https://mytesturl.com/123456789012/reset_queue",
			GetAccount: &db.Account{
				ID:               "1234567890",
				AdminRoleArn:     "arn:aws:iam::123456789012:role/AdminRole",
				PrincipalRoleArn: "arn:aws:iam::123456789012:role/PrincipalRole",
			},
			ExpectedBuildCount: 5,
			ExpectedBuildEnv: map[string]string{
				"RESET_ACCOUNT":                     "1234567890",
				"RESET_ACCOUNT_ADMIN_ROLE_NAME":     "AdminRole",
				"RESET_ACCOUNT_PRINCIPAL_ROLE_NAME": "PrincipalRole",
			},
			ExpectedOutput: &ResetOutput{
				Success: true,
				Accounts: map[string]ResetResult{
					"accountId-1": ResetResult{
						BuildTrigger:    true,
						MessageDeletion: true,
					},
					"accountId-2": ResetResult{
						BuildTrigger:    true,
						MessageDeletion: true,
					},
					"accountId-3": ResetResult{
						BuildTrigger:    true,
						MessageDeletion: true,
					},
					"accountId-4": ResetResult{
						BuildTrigger:    true,
						MessageDeletion: true,
					},
					"accountId-5": ResetResult{
						BuildTrigger:    true,
						MessageDeletion: true,
					},
				},
			},
			ExpectedError: nil,
		},
		// Test with GetAccount Error
		{
			ResetQueue:         createMockQueue(1),
			ResetQueueURL:      "https://mytesturl.com/123456789012/reset_queue",
			GetAccountError:    errors.New("Fail to Get Account"),
			ExpectedBuildCount: 0,
			ExpectedOutput: &ResetOutput{
				Success: false,
				Accounts: map[string]ResetResult{
					"accountId-1": ResetResult{
						BuildTrigger:    false,
						MessageDeletion: false,
					},
				},
			},
			ExpectedError: errors.New("Error: Could not successfully trigger a reset on all accounts"),
		},
		// Test with Invalid Admin Role Arn
		{
			ResetQueue:    createMockQueue(1),
			ResetQueueURL: "https://mytesturl.com/123456789012/reset_queue",
			GetAccount: &db.Account{
				AdminRoleArn: "MyArn",
			},
			ExpectedBuildCount: 0,
			ExpectedOutput: &ResetOutput{
				Success: false,
				Accounts: map[string]ResetResult{
					"accountId-1": ResetResult{
						BuildTrigger:    false,
						MessageDeletion: false,
					},
				},
			},
			ExpectedError: errors.New("Error: Could not successfully trigger a reset on all accounts"),
		},
		// Test with Message Receive Failure
		{
			ResetQueue:    createMockQueue(1),
			ResetQueueURL: "https://mytesturl.com/123456789012/fail_receive",
			GetAccount: &db.Account{
				AdminRoleArn: "arn:aws:iam::123456789012:role/AdminRole",
			},
			ExpectedBuildCount: 0,
			ExpectedOutput: &ResetOutput{
				Success:  false,
				Accounts: map[string]ResetResult{},
			},
			ExpectedError: errors.New("Error: Fail to Receive Message"),
		},
		// Test with Message Delete Failure
		{
			ResetQueue:    createMockQueue(1),
			ResetQueueURL: "https://mytesturl.com/123456789012/fail_delete",
			GetAccount: &db.Account{
				ID:               "1234567890",
				AdminRoleArn:     "arn:aws:iam::123456789012:role/AdminRole",
				PrincipalRoleArn: "arn:aws:iam::123456789012:role/PrincipalRole",
			},
			ExpectedBuildCount: 1,
			ExpectedBuildEnv: map[string]string{
				"RESET_ACCOUNT":                     "1234567890",
				"RESET_ACCOUNT_ADMIN_ROLE_NAME":     "AdminRole",
				"RESET_ACCOUNT_PRINCIPAL_ROLE_NAME": "PrincipalRole",
			},
			ExpectedOutput: &ResetOutput{
				Success: false,
				Accounts: map[string]ResetResult{
					"accountId-1": ResetResult{
						BuildTrigger:    true,
						MessageDeletion: false,
					},
				},
			},
			ExpectedError: errors.New("Error: Could not successfully trigger a " +
				"reset on all accounts"),
		},
		// Test with Trigger Build Failure
		{
			ResetQueue:    createMockQueue(1),
			ResetQueueURL: "https://mytesturl.com/123456789012/reset_queue",
			BuildError:    errors.New("Fail Triggering Build"),
			GetAccount: &db.Account{
				ID:               "1234567890",
				AdminRoleArn:     "arn:aws:iam::123456789012:role/AdminRole",
				PrincipalRoleArn: "arn:aws:iam::123456789012:role/PrincipalRole",
			},
			ExpectedBuildCount: 1,
			ExpectedBuildEnv: map[string]string{
				"RESET_ACCOUNT":                     "1234567890",
				"RESET_ACCOUNT_ADMIN_ROLE_NAME":     "AdminRole",
				"RESET_ACCOUNT_PRINCIPAL_ROLE_NAME": "PrincipalRole",
			},
			ExpectedOutput: &ResetOutput{
				Success: false,
				Accounts: map[string]ResetResult{
					"accountId-1": ResetResult{
						BuildTrigger:    false,
						MessageDeletion: false,
					},
				},
			},
			ExpectedError: errors.New("Error: Could not successfully trigger a " +
				"reset on all accounts"),
		},
	}
	buildName := "test"

	t.Run("Inputs/Output", func(t *testing.T) {
		// Iterate through each test in the list
		for _, test := range tests {
			// Mock the Database to return test.GetAccount
			mockDb := &dbMocks.DBer{}
			mockDb.
				On("GetAccount", mock.Anything).
				Return(test.GetAccount, test.GetAccountError)

			// Mock CodeBuild.StartBuild
			mockBuilder := &comMocks.Builder{}
			mockBuilder.On("StartBuild", mock.Anything, test.ExpectedBuildEnv).
				Return("mock-build-id", test.BuildError)

			// Set up the ResetInput
			resetInput := &ResetInput{
				ResetQueue:    test.ResetQueue,
				ResetQueueURL: &test.ResetQueueURL,
				ResetBuild:    mockBuilder,
				BuildName:     &buildName,
				DbSvc:         mockDb,
			}

			// Call the Reset function with the Mocked Queue and Pipeline
			resetOutput, err := Reset(resetInput)

			// Assert that ResetOutput and err was expected
			require.Equal(t, test.ExpectedOutput, resetOutput)
			require.Equal(t, test.ExpectedError, err)

			// Assert that mockBuild was called as expected
			mockBuilder.AssertNumberOfCalls(t, "StartBuild", test.ExpectedBuildCount)
		}
	})

	t.Run("Should set Lease.Status=ResetLock in DB, if the account has active leases", func(t *testing.T) {
		// Mock the DB Service
		mockDb := &dbMocks.DBer{}
		defer mockDb.AssertExpectations(t)

		// Mock leases for our Account
		// Should set Lease.Status=ResetLock
		// on the active lease (principalId-1)
		mockDb.On("GetAccount", mock.Anything).
			Return(mockAccount(), nil)

		// Mock the Build
		mockBuild := &comMocks.Builder{}
		mockBuild.On("StartBuild", mock.Anything, mock.Anything).
			Return("123", nil)

		// Call Reset
		queueURL := "https://mytesturl.com/123456789012/reset_queue"
		resetOutput, err := Reset(&ResetInput{
			// Will provide account "accountId-1"
			ResetQueue:    createMockQueue(1),
			ResetQueueURL: &queueURL,
			ResetBuild:    mockBuild,
			BuildName:     &buildName,
			DbSvc:         mockDb,
		})
		require.Nil(t, err)
		require.True(t, resetOutput.Success)
	})

	t.Run("Should not attempt to set the Lease status, if the account has no leases", func(t *testing.T) {
		// Mock the DB Service
		mockDb := &dbMocks.DBer{}
		defer mockDb.AssertExpectations(t)

		// Mock the DB to return no leases for the account
		mockDb.On("GetAccount", mock.Anything).
			Return(mockAccount(), nil)

		// Mock the build, and assert that we do _not_ call it
		mockBuild := comMocks.Builder{}
		mockBuild.On("StartBuild", mock.Anything, mock.Anything).
			Return("123", nil)
		defer mockBuild.AssertNotCalled(t, "StartBuild")

		// Call Reset
		queueURL := "https://mytesturl.com/123456789012/reset_queue"
		resetOutput, err := Reset(&ResetInput{
			// Will provide account "accountId-1"
			ResetQueue:    createMockQueue(1),
			ResetQueueURL: &queueURL,
			ResetBuild:    &mockBuild,
			BuildName:     &buildName,
			DbSvc:         mockDb,
		})
		require.Nil(t, err)
		require.True(t, resetOutput.Success)

		// Check that we didn't attempt to change any
		// lease statuses
		mockDb.AssertNotCalled(t, "TransitionLeaseStatus")
	})

	t.Run("Should return errors from DB", func(t *testing.T) {
		// Mock the DB to fail on `FindLeasesByAccount`
		mockDb := &dbMocks.DBer{}
		defer mockDb.AssertExpectations(t)
		dbErr := errors.New("Error: Could not successfully trigger a reset on all accounts")
		mockDb.On("GetAccount", mock.Anything).
			Return(mockAccount(), dbErr)

		// Mock the Build
		mockBuild := &comMocks.Builder{}
		mockBuild.On("StartBuild", mock.Anything, mock.Anything).
			Return("123", nil)

		// Call Reset
		queueURL := "https://mytesturl.com/123456789012/reset_queue"
		_, err := Reset(&ResetInput{
			// Will provide account "accountId-1"
			ResetQueue:    createMockQueue(1),
			ResetQueueURL: &queueURL,
			ResetBuild:    mockBuild,
			BuildName:     &buildName,
			DbSvc:         mockDb,
		})

		// Check that we get back our error from the DB
		require.Equal(t, dbErr, err, "Reset should return the DB error")
	})

	t.Run("Should pass parameters to the build environment", func(t *testing.T) {
		// Mock the Database
		mockDb := &dbMocks.DBer{}
		mockDb.
			On("FindLeasesByAccount", mock.Anything).
			Return([]*db.Lease{}, nil)
		mockDb.
			On("GetAccount", "accountId-1").
			Return(&db.Account{
				ID:               "123456789012",
				AdminRoleArn:     "arn:aws:iam::123456789012:role/AdminRole",
				PrincipalRoleArn: "arn:aws:iam::123456789012:role/PrincipalRole",
			}, nil)

		// Mock the Builder
		mockBuilder := &comMocks.Builder{}
		mockBuilder.On("StartBuild",
			aws.String("mock-build-name"),
			map[string]string{
				"RESET_ACCOUNT":                     "123456789012",
				"RESET_ACCOUNT_ADMIN_ROLE_NAME":     "AdminRole",
				"RESET_ACCOUNT_PRINCIPAL_ROLE_NAME": "PrincipalRole",
			},
		).
			Return("mock-build-id", nil)

		// Call the Reset function with the Mocked Queue and Pipeline
		_, err := Reset(&ResetInput{
			ResetQueue:    createMockQueue(1),
			ResetQueueURL: aws.String("https://mytesturl.com/123456789012/reset_queue"),
			ResetBuild:    mockBuilder,
			BuildName:     aws.String("mock-build-name"),
			DbSvc:         mockDb,
		})
		require.Nil(t, err)

		// Make sure we called builder.StartBuild, with the expected params
		mockBuilder.AssertExpectations(t)
	})
}

func TestExtractRoleNameFromARN(t *testing.T) {

	t.Run("should extract name from valid ARN", func(t *testing.T) {
		roleName, err := extractRoleNameFromARN(
			"arn:aws:iam::123456789012:role/myRole",
		)
		require.Nil(t, err)
		require.Equal(t, "myRole", roleName)
	})

	t.Run("should fail for invalid ARN", func(t *testing.T) {
		_, err := extractRoleNameFromARN("invalid_arn")
		require.NotNil(t, err)
		require.Equal(t, "Invalid Role ARN: invalid_arn", err.Error())
	})
}

func mockAccount() *db.Account {
	return &db.Account{
		ID:               "123456789012",
		AdminRoleArn:     "arn:aws:iam::123456789012:role/AdminRole",
		PrincipalRoleArn: "arn:aws:iam::123456789012:role/PrincipalRole",
	}
}
