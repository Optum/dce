package main

import (
	"testing"

	"github.com/pkg/errors"

	commock "github.com/Optum/Dcs/pkg/common/mocks"
	"github.com/Optum/Dcs/pkg/db"
	dbmock "github.com/Optum/Dcs/pkg/db/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testTransitionFinanceLockInput is the structured input for testing the helper
// function transitionFinanceLock
type testTransitionFinanceLockInput struct {
	ExpectedError              error
	FindLeasesByAccount        []*db.DcsLease
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
			FindLeasesByAccount: []*db.DcsLease{
				{
					AccountID:   "123",
					PrincipalID: "abc",
					LeaseStatus: "FinanceLock",
				},
			},
		},
		// Happy Path No FinanceLock
		{
			FindLeasesByAccount: []*db.DcsLease{
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
			FindLeasesByAccount: []*db.DcsLease{
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

// testEnqueueDcs is the structured input for testing the function
// enqueueDcs
type testEnqueueDcsesInput struct {
	ExpectedError            error
	SendMessageError         error
	FindLeasesByAccountError error
}

// TestEnqueueDcs tests and verifies the flow of adding all dcs accounts
// provided into the reset queue and transition the finance lock if necessary
func TestEnqueueDcs(t *testing.T) {
	// Construct test scenarios
	tests := []testEnqueueDcsesInput{
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
	dcses := []*db.DcsAccount{
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
			[]*db.DcsLease{}, test.FindLeasesByAccountError)

		// Call enqueueDcses
		err := enqueueDcses(dcses, &queueURL, &mockQueue, &mockDB)

		// Assert expectations
		if test.ExpectedError != nil {
			require.Equal(t, test.ExpectedError.Error(), err.Error())
		} else {
			require.Nil(t, err)
		}
	}
}
