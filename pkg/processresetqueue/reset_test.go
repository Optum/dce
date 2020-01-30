package processresetqueue

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/stretchr/testify/mock"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/arn"
	comMocks "github.com/Optum/dce/pkg/common/mocks"
	"github.com/stretchr/testify/require"

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
		accountID := fmt.Sprintf("12345678901%d", queue.NumLoop)
		acct := &account.Account{
			ID:               &accountID,
			Status:           account.StatusNotReady.StatusPtr(),
			AdminRoleArn:     arn.New("aws", "iam", "", accountID, "role/AdminRole"),
			PrincipalRoleArn: arn.New("aws", "iam", "", accountID, "role/PrincipalRole"),
		}
		body, err := json.Marshal(acct)
		if err != nil {
			return nil, err
		}
		receiptHandle := receiptHandle
		messageID := "messageId"
		messages = append(messages, &sqs.Message{
			Body:          aws.String(string(body)),
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
	Name            string
	QueueLength     int
	BuildLength     int
	ResetQueueURL   string
	BuildError      error
	GetAccountError error
	ExpectedOutput  *ResetOutput
	ExpectedError   error
}

// TestReset verifies the Reset function works as intended, where it should be
// able to digest from a Queue and trigger the respective Pipelines, while
// returning a correct ResetOutput response.
func TestReset(t *testing.T) {

	// Set up the list of tests to execute
	tests := []resetTest{
		// Test with No Messages Received
		{
			Name:          "When the queue is empty.  Process nothing with success.",
			QueueLength:   0,
			BuildLength:   0,
			ResetQueueURL: "https://mytesturl.com/123456789012/reset_queue",
			ExpectedOutput: &ResetOutput{
				Success:  true,
				Accounts: map[string]ResetResult{},
			},
			ExpectedError: nil,
		},
		// Test with 1 Message Received
		{
			Name:          "When the queue has 1 record.  Process 1 record with success.",
			QueueLength:   1,
			BuildLength:   1,
			ResetQueueURL: "https://mytesturl.com/123456789012/reset_queue",
			ExpectedOutput: &ResetOutput{
				Success: true,
				Accounts: map[string]ResetResult{
					"123456789011": ResetResult{
						BuildTrigger:    true,
						MessageDeletion: true,
					},
				},
			},
			ExpectedError: nil,
		},
		// Test with 5 Message Received
		{
			Name:          "When the queue has 5 records.  Process 5 records with success.",
			QueueLength:   5,
			BuildLength:   5,
			ResetQueueURL: "https://mytesturl.com/123456789012/reset_queue",
			ExpectedOutput: &ResetOutput{
				Success: true,
				Accounts: map[string]ResetResult{
					"123456789011": ResetResult{
						BuildTrigger:    true,
						MessageDeletion: true,
					},
					"123456789012": ResetResult{
						BuildTrigger:    true,
						MessageDeletion: true,
					},
					"123456789013": ResetResult{
						BuildTrigger:    true,
						MessageDeletion: true,
					},
					"123456789014": ResetResult{
						BuildTrigger:    true,
						MessageDeletion: true,
					},
					"123456789015": ResetResult{
						BuildTrigger:    true,
						MessageDeletion: true,
					},
				},
			},
			ExpectedError: nil,
		},
		// Test with Message Receive Failure
		{
			Name:          "When the queue has 1 message and their is a failure recieving the message.  Retun the appropriate error.",
			QueueLength:   1,
			BuildLength:   0,
			ResetQueueURL: "https://mytesturl.com/123456789012/fail_receive",
			ExpectedOutput: &ResetOutput{
				Success:  false,
				Accounts: map[string]ResetResult{},
			},
			ExpectedError: errors.New("Error: Fail to Receive Message"),
		},
		// Test with Message Delete Failure
		{
			Name:          "When the queue has 1 message and their is a failure deleting the message.  Retun the appropriate error.",
			QueueLength:   1,
			BuildLength:   1,
			ResetQueueURL: "https://mytesturl.com/123456789012/fail_delete",
			ExpectedOutput: &ResetOutput{
				Success: false,
				Accounts: map[string]ResetResult{
					"123456789011": ResetResult{
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
			Name:          "When the queue has 1 message and their is a failure triggering code build.  Retun the appropriate error.",
			QueueLength:   1,
			BuildLength:   1,
			ResetQueueURL: "https://mytesturl.com/123456789012/reset_queue",
			BuildError:    errors.New("Fail Triggering Build"),
			ExpectedOutput: &ResetOutput{
				Success: false,
				Accounts: map[string]ResetResult{
					"123456789011": ResetResult{
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

	// Iterate through each test in the list
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// Mock CodeBuild.StartBuild
			mockBuilder := &comMocks.Builder{}
			for i := 1; i <= test.BuildLength; i++ {
				buildEnv := map[string]string{
					"RESET_ACCOUNT":                     fmt.Sprintf("12345678901%d", i),
					"RESET_ACCOUNT_ADMIN_ROLE_NAME":     "AdminRole",
					"RESET_ACCOUNT_PRINCIPAL_ROLE_NAME": "PrincipalRole",
				}
				mockBuilder.On("StartBuild", mock.Anything, buildEnv).
					Return("mock-build-id", test.BuildError)
			}

			mockQueue := createMockQueue(test.QueueLength)
			// Set up the ResetInput
			resetInput := &ResetInput{
				ResetQueue:    mockQueue,
				ResetQueueURL: &test.ResetQueueURL,
				ResetBuild:    mockBuilder,
				BuildName:     &buildName,
			}

			// Call the Reset function with the Mocked Queue and Pipeline
			resetOutput, err := Reset(resetInput)

			// Assert that ResetOutput and err was expected
			require.Equal(t, test.ExpectedOutput, resetOutput)
			require.Equal(t, test.ExpectedError, err)

			// Assert that mockBuild was called as expected
			mockBuilder.AssertExpectations(t)
		})
	}
}
