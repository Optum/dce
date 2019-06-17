package main

import (
	"testing"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commock "github.com/Optum/Redbox/pkg/common/mocks"
	"github.com/Optum/Redbox/pkg/db"
	dbmock "github.com/Optum/Redbox/pkg/db/mocks"
)

// testTransitionFinanceLockInput is the structured input for testing the helper
// function transitionFinanceLock
type testTransitionFinanceLockInput struct {
	ExpectedError                       error
	FindAssignmentsByAccountAssignments []*db.RedboxAccountAssignment
	FindAssignmentsByAccountError       error
	TransitionAssignmentStatusError     error
}

// TestTransitionFinanceLock tests and verifies the flow of getting the
// Account Assignments and transitioning a FinanceLock status to Active in
// transitionFinanceLock
func TestTransitionFinanceLock(t *testing.T) {
	// Construct test scenarios
	tests := []testTransitionFinanceLockInput{
		// Happy Path FinanceLock
		{
			FindAssignmentsByAccountAssignments: []*db.RedboxAccountAssignment{
				{
					AccountID:        "123",
					UserID:           "abc",
					AssignmentStatus: "FinanceLock",
				},
			},
		},
		// Happy Path No FinanceLock
		{
			FindAssignmentsByAccountAssignments: []*db.RedboxAccountAssignment{
				{
					AccountID:        "123",
					UserID:           "abc",
					AssignmentStatus: "Active",
				},
			},
		},
		// Happy Path No Assignments
		{},
		// FindAssignmentsByAccount Failure
		{
			ExpectedError:                 errors.New("FindAssignments Fail"),
			FindAssignmentsByAccountError: errors.New("FindAssignments Fail"),
		},
		// TransitionAssignmentStatus Failure
		{
			ExpectedError: errors.New("Transition Fail"),
			FindAssignmentsByAccountAssignments: []*db.RedboxAccountAssignment{
				{
					AccountID:        "123",
					UserID:           "abc",
					AssignmentStatus: "FinanceLock",
				},
			},
			TransitionAssignmentStatusError: errors.New("Transition Fail"),
		},
	}

	// Iterate through each test in the list
	account := "123"
	for _, test := range tests {
		// Setup mocks
		mockDB := dbmock.DBer{}
		mockDB.On("FindAssignmentsByAccount", mock.Anything).Return(
			test.FindAssignmentsByAccountAssignments,
			test.FindAssignmentsByAccountError)
		mockDB.On("TransitionAssignmentStatus", mock.Anything, mock.Anything,
			mock.Anything, mock.Anything).Return(nil,
			test.TransitionAssignmentStatusError)

		// Call transitionFinanceLock
		err := transitionFinanceLock(account, &mockDB)

		// Assert expectations
		if test.ExpectedError != nil {
			require.Equal(t, test.ExpectedError.Error(), err.Error())
		} else {
			require.Nil(t, err)
		}
	}
}

// testEnqueueRedbox is the structured input for testing the function
// enqueueRedbox
type testEnqueueRedboxesInput struct {
	ExpectedError                 error
	SendMessageError              error
	FindAssignmentsByAccountError error
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
		// TransitionFinanceLockFailure
		{
			ExpectedError: errors.Wrap(errors.New("Find Assignments Fail"),
				"Failed to enqueue accounts"),
			FindAssignmentsByAccountError: errors.New("Find Assignments Fail"),
		},
	}

	// Iterate through each test in the list
	redboxes := []*db.RedboxAccount{
		{
			ID:            "123",
			AccountStatus: "Assigned",
		},
	}
	queueURL := "url"
	for _, test := range tests {
		// Setup mocks
		mockQueue := commock.Queue{}
		mockQueue.On("SendMessage", mock.Anything, mock.Anything).Return(
			test.SendMessageError)

		mockDB := dbmock.DBer{}
		mockDB.On("FindAssignmentsByAccount", mock.Anything).Return(
			[]*db.RedboxAccountAssignment{}, test.FindAssignmentsByAccountError)

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
