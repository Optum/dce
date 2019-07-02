package main

import (
	"errors"
	"testing"

	"github.com/Optum/Redbox/pkg/db"
	dbmock "github.com/Optum/Redbox/pkg/db/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testUpdateAssignmentInput is the structure input used for table driven
// testing for Finance Lock of Account
type testUpdateAssignmentInput struct {
	AccountInput                      string
	ExpectedError                     error
	FindAssignmentsByAccount          []*db.RedboxAccountAssignment
	FindAssignmentsByAccountError     error
	TransitionAssignmentStatusError   error
	TransitionAssignmentStatus        *db.RedboxAccountAssignment
	TransitionAssignmentStatusAccount *db.RedboxAccountAssignment
}

func TestUpdateAssignment(t *testing.T) {
	// Construct test scenarios
	tests := []testUpdateAssignmentInput{
		// Happy Path
		{
			AccountInput:  "123",
			ExpectedError: nil,
			FindAssignmentsByAccount: []*db.RedboxAccountAssignment{
				{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Active,
				},
			},
			TransitionAssignmentStatusAccount: &db.RedboxAccountAssignment{
				UserID:           "abc",
				AccountID:        "123",
				AssignmentStatus: db.Active,
			},
		},
		// No assigned account
		{
			AccountInput: "123",
			FindAssignmentsByAccount: []*db.RedboxAccountAssignment{
				{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Active,
				},
			},
			TransitionAssignmentStatusAccount: &db.RedboxAccountAssignment{
				UserID:           "abc",
				AccountID:        "123",
				AssignmentStatus: db.Active,
			},
			ExpectedError:                 errors.New("Failed to find Assignment for user"),
			FindAssignmentsByAccountError: errors.New("Failed to find Assignment for user"),
		},
		// TransitionAssignment Error
		{
			AccountInput: "123",
			FindAssignmentsByAccount: []*db.RedboxAccountAssignment{
				{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Active,
				},
			},
			TransitionAssignmentStatusAccount: &db.RedboxAccountAssignment{
				UserID:           "abc",
				AccountID:        "123",
				AssignmentStatus: db.Active,
			},
			ExpectedError:                   errors.New("Failed to transition Assignment Status"),
			TransitionAssignmentStatusError: errors.New("Failed to transition Assignment Status"),
		},
	}

	for _, test := range tests {
		// Setup mocks
		mockDB := &dbmock.DBer{}
		mockDB.On("FindAssignmentsByAccount", mock.Anything).Return(
			test.FindAssignmentsByAccount,
			test.FindAssignmentsByAccountError)
		mockDB.On("TransitionAssignmentStatus", mock.Anything, mock.Anything,
			mock.Anything, mock.Anything).Return(test.TransitionAssignmentStatusAccount,
			test.TransitionAssignmentStatusError)

		// Call FinanceLock
		err := updateAssignment(test.AccountInput, mockDB)

		// Assert that the expected output is correct
		require.Equal(t, test.TransitionAssignmentStatusAccount.AccountID, "123")
		require.Equal(t, test.TransitionAssignmentStatusAccount.UserID, "abc")
		require.Equal(t, test.FindAssignmentsByAccount[0].UserID, "abc")
		require.Equal(t, test.FindAssignmentsByAccount[0].AccountID, "123")
		require.Equal(t, test.ExpectedError, err)
	}
}
