package main

import (
	"testing"

	"github.com/pkg/errors"

	commock "github.com/Optum/Redbox/pkg/common/mocks"
	"github.com/Optum/Redbox/pkg/db"
	dbmock "github.com/Optum/Redbox/pkg/db/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testEnqueueRedbox is the structured input for testing the function
// enqueueRedbox
type testEnqueueRedboxesInput struct {
	ExpectedError            error
	SendMessageError         error
	FindLeasesByAccountError error
}

// TestEnqueueRedbox tests and verifies the flow of adding all redbox accounts
// provided into the reset queue and transition the finance lock if necessary
func TestEnqueueRedbox(t *testing.T) {
	// Construct test scenarios
	tests := []testEnqueueRedboxesInput{
		// Happy Path
		{},
		// SendMessage Failure
		{
			ExpectedError: errors.Wrap(errors.New("Send Message Fail"),
				"Failed to enqueue accounts"),
			SendMessageError: errors.New("Send Message Fail"),
		},
	}

	// Iterate through each test in the list
	redboxes := []*db.RedboxAccount{
		{
			ID:            "123",
			AccountStatus: "Leased",
		},
	}
	queueURL := "url"
	for _, test := range tests {
		// Setup mocks
		mockQueue := commock.Queue{}
		mockQueue.On("SendMessage", mock.Anything, mock.Anything).Return(
			test.SendMessageError)

		mockDB := dbmock.DBer{}
		// Call enqueueRedboxes
		err := enqueueRedboxes(redboxes, &queueURL, &mockQueue, &mockDB)

		// Assert expectations
		if test.ExpectedError != nil {
			require.Equal(t, test.ExpectedError.Error(), err.Error())
		} else {
			require.Nil(t, err)
		}
	}
}
