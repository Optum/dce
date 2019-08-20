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

// testTransitionFinanceLockInput is the structured input for testing the helper
// function transitionFinanceLock
type testTransitionFinanceLockInput struct {
	ExpectedError              error
	FindLeasesByAccount        []*db.RedboxLease
	FindLeasesByAccountError   error
	TransitionLeaseStatusError error
}

// TestTransitionFinanceLock tests and verifies the flow of getting the
// Account Leases and transitioning a FinanceLock status to Active in
// transitionFinanceLock
func TestTransitionFinanceLock(t *testing.T) {
	// Construct test scenarios
	tests := []testTransitionFinanceLockInput{
		// Happy Path FinanceLock
		{
			FindLeasesByAccount: []*db.RedboxLease{
				{
					AccountID:   "123",
					PrincipalID: "abc",
					LeaseStatus: "FinanceLock",
				},
			},
		},
		// Happy Path No FinanceLock
		{
			FindLeasesByAccount: []*db.RedboxLease{
				{
					AccountID:   "123",
					PrincipalID: "abc",
					LeaseStatus: "Active",
				},
			},
		},
		// Happy Path No Leases
		{},
		// FindLeasesByAccount Failure
		{
			ExpectedError:            errors.New("FindLeases Fail"),
			FindLeasesByAccountError: errors.New("FindLeases Fail"),
		},
		// TransitionLeaseStatus Failure
		{
			ExpectedError: errors.New("Transition Fail"),
			FindLeasesByAccount: []*db.RedboxLease{
				{
					AccountID:   "123",
					PrincipalID: "abc",
					LeaseStatus: "FinanceLock",
				},
			},
			TransitionLeaseStatusError: errors.New("Transition Fail"),
		},
	}

	// Iterate through each test in the list
	account := "123"
	for _, test := range tests {
		// Setup mocks
		mockDB := dbmock.DBer{}
		mockDB.On("FindLeasesByAccount", mock.Anything).Return(
			test.FindLeasesByAccount,
			test.FindLeasesByAccountError)
		mockDB.On("TransitionLeaseStatus", mock.Anything, mock.Anything,
			mock.Anything, mock.Anything).Return(nil,
			test.TransitionLeaseStatusError)

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
		// TransitionFinanceLockFailure
		{
			ExpectedError: errors.Wrap(errors.New("Find Leases Fail"),
				"Failed to enqueue accounts"),
			FindLeasesByAccountError: errors.New("Find Leases Fail"),
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
		mockDB.On("FindLeasesByAccount", mock.Anything).Return(
			[]*db.RedboxLease{}, test.FindLeasesByAccountError)

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
