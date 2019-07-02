package trigger

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"

	comMocks "github.com/Optum/Redbox/pkg/common/mocks"
	"github.com/Optum/Redbox/pkg/db"
	dbMocks "github.com/Optum/Redbox/pkg/db/mocks"
	"github.com/stretchr/testify/require"

	"github.com/Optum/Redbox/pkg/common"
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
	TestQueue  common.Queue
	TestURL    string
	BuildID    string
	BuildError error
	Output     *ResetOutput
	Error      error
}

// TestReset verifies the Reset function works as intended, where it should be
// able to digest from a Queue and trigger the respective Pipelines, while
// returning a correct ResetOutput response.
func TestReset(t *testing.T) {

	// Set up the list of tests to execute
	tests := []resetTest{
		// Test with No Messages Received
		{
			TestQueue: createMockQueue(0),
			TestURL:   "https://mytesturl.com/123456789012/reset_queue",
			BuildID:   "123",
			Output: &ResetOutput{
				Success:  true,
				Accounts: map[string]ResetResult{},
			},
			Error: nil,
		},
		// Test with 1 Message Received
		{
			TestQueue: createMockQueue(1),
			TestURL:   "https://mytesturl.com/123456789012/reset_queue",
			BuildID:   "123",
			Output: &ResetOutput{
				Success: true,
				Accounts: map[string]ResetResult{
					"accountId-1": ResetResult{
						BuildTrigger:    true,
						MessageDeletion: true,
					},
				},
			},
			Error: nil,
		},
		// Test with 5 Message Received
		{
			TestQueue: createMockQueue(5),
			TestURL:   "https://mytesturl.com/123456789012/reset_queue",
			BuildID:   "123",
			Output: &ResetOutput{
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
			Error: nil,
		},
		// Test with Message Receive Failure
		{
			TestQueue: createMockQueue(1),
			TestURL:   "https://mytesturl.com/123456789012/fail_receive",
			BuildID:   "123",
			Output: &ResetOutput{
				Success:  false,
				Accounts: map[string]ResetResult{},
			},
			Error: errors.New("Error: Fail to Receive Message"),
		},
		// Test with Message Delete Failure
		{
			TestQueue: createMockQueue(1),
			TestURL:   "https://mytesturl.com/123456789012/fail_delete",
			BuildID:   "123",
			Output: &ResetOutput{
				Success: false,
				Accounts: map[string]ResetResult{
					"accountId-1": ResetResult{
						BuildTrigger:    true,
						MessageDeletion: false,
					},
				},
			},
			Error: errors.New("Error: Could not successfully trigger a " +
				"reset on all accounts"),
		},
		// Test with Trigger Build Failure
		{
			TestQueue:  createMockQueue(1),
			TestURL:    "https://mytesturl.com/123456789012/reset_queue",
			BuildError: errors.New("Fail Triggering Build"),
			Output: &ResetOutput{
				Success: false,
				Accounts: map[string]ResetResult{
					"accountId-1": ResetResult{
						BuildTrigger:    false,
						MessageDeletion: false,
					},
				},
			},
			Error: errors.New("Error: Could not successfully trigger a " +
				"reset on all accounts"),
		},
	}
	buildName := "test"

	t.Run("Inputs/Output", func(t *testing.T) {
		// Iterate through each test in the list
		for _, test := range tests {
			// Mock the Database
			mockDb := &dbMocks.DBer{}
			mockDb.
				On(
					"FindAssignmentsByAccount",
					mock.MatchedBy(func(accountID string) bool { return true }),
				).
				Return([]*db.RedboxAccountAssignment{}, nil)

			// Mock the Build
			mockBuild := &comMocks.Builder{}
			mockBuild.On("StartBuild", mock.Anything, mock.Anything).Return(
				test.BuildID, test.BuildError)

			// Set up the ResetInput
			resetInput := &ResetInput{
				ResetQueue:    test.TestQueue,
				ResetQueueURL: &test.TestURL,
				ResetBuild:    mockBuild,
				BuildName:     &buildName,
				DbSvc:         mockDb,
			}

			// Call the Reset function with the Mocked Queue and Pipeline
			resetOutput, err := Reset(resetInput)

			// Assert that ResetOutput and err was expected
			require.Equal(t, test.Output, resetOutput)
			require.Equal(t, test.Error, err)
		}
	})

	t.Run("Should set Assignment.Status=ResetFinanceLock in DB, if the account has Active assignments", func(t *testing.T) {
		// Mock the DB Service
		mockDb := &dbMocks.DBer{}
		defer mockDb.AssertExpectations(t)

		// Mock assignments for our Account
		mockDb.
			On("FindAssignmentsByAccount", "accountId-1").
			Return([]*db.RedboxAccountAssignment{
				{
					UserID:           "userId-1",
					AssignmentStatus: db.Decommissioned,
				},
				{
					UserID:           "userId-2",
					AssignmentStatus: db.FinanceLock,
				},
			}, nil)

		// Should set Assignment.Status=ResetFinanceLock
		// on the FinanceLock assignment (userId-2)
		mockDb.
			On(
				"TransitionAssignmentStatus",
				"accountId-1", "userId-2",
				db.FinanceLock, db.ResetFinanceLock,
			).
			Return(&db.RedboxAccountAssignment{}, nil)

		// Mock the Build
		mockBuild := &comMocks.Builder{}
		mockBuild.On("StartBuild", mock.Anything, mock.Anything).Return(
			"123", nil)

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

	t.Run("Should set Assignment.Status=ResetLock in DB, if the account has active assignments", func(t *testing.T) {
		// Mock the DB Service
		mockDb := &dbMocks.DBer{}
		defer mockDb.AssertExpectations(t)

		// Mock assignments for our Account
		mockDb.
			On("FindAssignmentsByAccount", "accountId-1").
			Return([]*db.RedboxAccountAssignment{
				{
					UserID:           "userId-1",
					AssignmentStatus: db.Active,
				},
				{
					UserID:           "userId-2",
					AssignmentStatus: db.Decommissioned,
				},
			}, nil)

		// Should set Assignment.Status=ResetLock
		// on the active assignment (userId-1)
		mockDb.
			On(
				"TransitionAssignmentStatus",
				"accountId-1", "userId-1",
				db.Active, db.ResetLock,
			).
			Return(&db.RedboxAccountAssignment{}, nil)

		// Mock the Build
		mockBuild := &comMocks.Builder{}
		mockBuild.On("StartBuild", mock.Anything, mock.Anything).Return(
			"123", nil)

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

	t.Run("Should not attempt to set the Assignment status, if the account has no assignments", func(t *testing.T) {
		// Mock the DB Service
		mockDb := &dbMocks.DBer{}
		defer mockDb.AssertExpectations(t)

		// Mock the DB to return no assignments for the account
		mockDb.
			On("FindAssignmentsByAccount", "accountId-1").
			Return([]*db.RedboxAccountAssignment{}, nil)

		// Mock the build, and assert that we do _not_ call it
		mockBuild := comMocks.Builder{}
		mockBuild.On("StartBuild", mock.Anything, mock.Anything).Return(
			"123", nil)
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
		// assignment statuses
		mockDb.AssertNotCalled(t, "TransitionAssignmentStatus")
	})

	t.Run("Should not trigger pipeline if the assignment status update fails", func(t *testing.T) {
		// Mock the DB to fail on `FindAssignmentsByAccount`
		mockDb := &dbMocks.DBer{}
		defer mockDb.AssertExpectations(t)
		dbErr := errors.New("Error: Could not successfully trigger a reset on all accounts")
		mockDb.
			On("FindAssignmentsByAccount", "accountId-1").
			Return(nil, dbErr)

		// Mock the build, and assert that we do _not_ call it
		mockBuild := comMocks.Builder{}
		defer mockBuild.AssertNotCalled(t, "StartBuild")

		// Call Reset
		queueURL := "https://mytesturl.com/123456789012/reset_queue"
		_, err := Reset(&ResetInput{
			// Will provide account "accountId-1"
			ResetQueue:    createMockQueue(1),
			ResetQueueURL: &queueURL,
			ResetBuild:    &mockBuild,
			BuildName:     &buildName,
			DbSvc:         mockDb,
		})
		require.NotNil(t, err)
	})

	t.Run("Should return errors from DB", func(t *testing.T) {
		// Mock the DB to fail on `FindAssignmentsByAccount`
		mockDb := &dbMocks.DBer{}
		defer mockDb.AssertExpectations(t)
		dbErr := errors.New("Error: Could not successfully trigger a reset on all accounts")
		mockDb.
			On("FindAssignmentsByAccount", "accountId-1").
			Return(nil, dbErr)

		// Mock the Build
		mockBuild := &comMocks.Builder{}
		mockBuild.On("StartBuild", mock.Anything, mock.Anything).Return(
			"123", nil)

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
}
