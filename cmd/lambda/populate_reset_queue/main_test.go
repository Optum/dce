package main

import (
	"testing"

	"github.com/pkg/errors"

	commock "github.com/Optum/Dce/pkg/common/mocks"
	"github.com/Optum/Dce/pkg/db"
	dbmock "github.com/Optum/Dce/pkg/db/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testTransitionFinanceLockInput is the structured input for testing the helper
// function transitionFinanceLock
type testTransitionFinanceLockInput struct {
	ExpectedError              error
	FindLeasesByAccount        []*db.DceLease
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
			FindLeasesByAccount: []*db.DceLease{
				{
					AccountID:   "123",
					PrincipalID: "abc",
					LeaseStatus: "FinanceLock",
				},
			},
		},
		// Happy Path No FinanceLock
		{
			FindLeasesByAccount: []*db.DceLease{
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
			FindLeasesByAccount: []*db.DceLease{
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

// testEnqueueDce is the structured input for testing the function
// enqueueDce
type testEnqueueDceesInput struct {
	ExpectedError            error
	SendMessageError         error
	FindLeasesByAccountError error
}

// TestEnqueueDce tests and verifies the flow of adding all dce accounts
// provided into the reset queue and transition the finance lock if necessary
func TestEnqueueDce(t *testing.T) {
	// Construct test scenarios
	tests := []testEnqueueDceesInput{
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
	dcees := []*db.DceAccount{
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
			[]*db.DceLease{}, test.FindLeasesByAccountError)

		// Call enqueueDcees
		err := enqueueDcees(dcees, &queueURL, &mockQueue, &mockDB)

		// Assert expectations
		if test.ExpectedError != nil {
			require.Equal(t, test.ExpectedError.Error(), err.Error())
		} else {
			require.Nil(t, err)
		}
	}
}
