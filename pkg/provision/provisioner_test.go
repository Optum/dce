package provision

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/db/mocks"
)

// testFindUserActiveAssignmentInput is the structure input used for table
// driven testing for FindUserActiveAssignment
type testFindUserActiveAssignmentInput struct {
	ExpectedError                   error
	ExpectedAssignmentUser          string
	ExpectedAssignmentAccount       string
	FindAssignmentByUserAssignments []*db.RedboxAccountAssignment
	FindAssignmentByUserError       error
	ExpectAssignment                bool
}

// TestFindUserActiveAssignment tests and verifies the flow of the helper
// function to find any active user assignments
func TestFindUserActiveAssignment(t *testing.T) {
	// Construct test scenarios
	tests := []testFindUserActiveAssignmentInput{
		// Happy Path - Decommissioned
		{
			ExpectedError:             nil,
			ExpectedAssignmentUser:    "",
			ExpectedAssignmentAccount: "",
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Decommissioned,
				},
			},
			ExpectAssignment: true,
		},
		// Happy Path - Active
		{
			ExpectedError:             nil,
			ExpectedAssignmentUser:    "abc",
			ExpectedAssignmentAccount: "123",
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Active,
				},
			},
			ExpectAssignment: true,
		},
		// Happy Path - FinanceLock
		{
			ExpectedError:             nil,
			ExpectedAssignmentUser:    "abc",
			ExpectedAssignmentAccount: "123",
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.FinanceLock,
				},
			},
			ExpectAssignment: true,
		},
		// Happy Path - ResetLock
		{
			ExpectedError:             nil,
			ExpectedAssignmentUser:    "abc",
			ExpectedAssignmentAccount: "123",
			FindAssignmentByUserAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.ResetLock,
				},
			},
			ExpectAssignment: true,
		},
		// Error FindAssignmentByUser
		{
			ExpectedError:             errors.New("Error Finding Assignment"),
			FindAssignmentByUserError: errors.New("Error Finding Assignment"),
		},
	}

	// Iterate through each test in the list
	user := "abc"
	for _, test := range tests {
		// Setup mocks
		mockDB := &mocks.DBer{}
		mockDB.On("FindAssignmentByUser", mock.Anything).Return(
			test.FindAssignmentByUserAssignments,
			test.FindAssignmentByUserError)

		// Create the AccountProvision
		prov := AccountProvision{
			DBSvc: mockDB,
		}

		// Call findUserActiveAssignment
		assignment, err := prov.FindUserActiveAssignment(user)

		// Assert that the expected output is correct
		require.Equal(t, test.ExpectedError, err)
		if test.ExpectAssignment {
			require.Equal(t, test.ExpectedAssignmentUser, assignment.UserID)
			require.Equal(t, test.ExpectedAssignmentAccount,
				assignment.AccountID)
		} else {
			require.Nil(t, assignment)
		}
	}
}

// testFindUserAssignmentWithAccountInput is the structure input used for table
// driven testing for FindUserAssignmentWithAccount
type testFindUserAssignmentWithAccountInput struct {
	ExpectedError                       error
	ExpectedAssignmentUser              string
	ExpectedAssignmentAccount           string
	FindAssignmentsByAccountAssignments []*db.RedboxAccountAssignment
	FindAssignmentsByAccountError       error
	ExpectAssignment                    bool
}

// TestFindUserAssignmentWithAccount tests and verifies the flow of the helper
// function to find an assignment between a user and an account
func TestFindUserAssignmentWithAccount(t *testing.T) {
	// Construct test scenarios
	tests := []testFindUserAssignmentWithAccountInput{
		// Happy Path - Assignment Exists
		{
			ExpectedError:             nil,
			ExpectedAssignmentUser:    "abc",
			ExpectedAssignmentAccount: "123",
			FindAssignmentsByAccountAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "abc",
					AccountID:        "123",
					AssignmentStatus: db.Decommissioned,
				},
			},
			ExpectAssignment: true,
		},
		// Happy Path - Assignment Does Not Exist
		{
			ExpectedError:             nil,
			ExpectedAssignmentUser:    "",
			ExpectedAssignmentAccount: "",
			FindAssignmentsByAccountAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "def",
					AccountID:        "123",
					AssignmentStatus: db.Decommissioned,
				},
			},
			ExpectAssignment: true,
		},
		// Error FindAssignmentsByAccount
		{
			ExpectedError:                 errors.New("Error Finding Assignment"),
			FindAssignmentsByAccountError: errors.New("Error Finding Assignment"),
		},
		// Error Account has Active Assignment
		{
			ExpectedError: errors.New("Attempt to Assign Active Account as " +
				"new Redbox - 123"),
			FindAssignmentsByAccountAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "def",
					AccountID:        "123",
					AssignmentStatus: db.Active,
				},
			},
		},
		// Error Account has FinaceLock Assignment
		{
			ExpectedError: errors.New("Attempt to Assign Active Account as " +
				"new Redbox - 123"),
			FindAssignmentsByAccountAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "def",
					AccountID:        "123",
					AssignmentStatus: db.FinanceLock,
				},
			},
		},
		// Error Account has ResetLock Assignment
		{
			ExpectedError: errors.New("Attempt to Assign Active Account as " +
				"new Redbox - 123"),
			FindAssignmentsByAccountAssignments: []*db.RedboxAccountAssignment{
				&db.RedboxAccountAssignment{
					UserID:           "def",
					AccountID:        "123",
					AssignmentStatus: db.ResetLock,
				},
			},
		},
	}

	// Iterate through each test in the list
	user := "abc"
	account := "123"
	for _, test := range tests {
		// Setup mocks
		mockDB := &mocks.DBer{}
		mockDB.On("FindAssignmentsByAccount", mock.Anything).Return(
			test.FindAssignmentsByAccountAssignments,
			test.FindAssignmentsByAccountError)

		// Create the AccountProvision
		prov := AccountProvision{
			DBSvc: mockDB,
		}

		// Call findUserAssignmentWithAccount
		assignment, err := prov.FindUserAssignmentWithAccount(user, account)

		// Assert that the expected output is correct
		require.Equal(t, test.ExpectedError, err)
		if test.ExpectAssignment {
			require.Equal(t, test.ExpectedAssignmentUser, assignment.UserID)
			require.Equal(t, test.ExpectedAssignmentAccount,
				assignment.AccountID)
		} else {
			require.Nil(t, assignment)
		}
	}
}

// testActivateAccountAssignmentInput is the structure input used for table
// driven testing for ActivateAccountAssignment
type testActivateAccountAssignmentInput struct {
	ExpectedAccountAssignment             *db.RedboxAccountAssignment
	ExpectedError                         error
	Create                                bool
	PutAccountAssignmentAccountAssignment *db.RedboxAccountAssignment
	PutAccountAssignmentError             error
	TransitionAssignmentStatusAssignment  *db.RedboxAccountAssignment
	TransitionAssignmentStatusError       error
}

// TestActivateAccountAssignment tests and verifies the flow of the helper
// function to create or update an account assignment as active for a user
func TestActivateAccountAssignment(t *testing.T) {
	// Construct test scenarios
	accountAssignment := &db.RedboxAccountAssignment{
		AccountID:        "123",
		UserID:           "abc",
		AssignmentStatus: db.Active,
	}
	tests := []testActivateAccountAssignmentInput{
		// Happy Path - Create
		{
			Create:                                true,
			ExpectedAccountAssignment:             accountAssignment,
			PutAccountAssignmentAccountAssignment: accountAssignment,
		},
		// Happy Path - Update
		{
			ExpectedAccountAssignment: accountAssignment,
			TransitionAssignmentStatusAssignment: &db.RedboxAccountAssignment{
				AccountID:        "123",
				UserID:           "abc",
				AssignmentStatus: db.Active,
				LastModifiedOn:   456,
			},
		},
		// Fail PutAccountAssignment
		{
			ExpectedError:             errors.New("Fail Creating New Assignment"),
			Create:                    true,
			PutAccountAssignmentError: errors.New("Fail Creating New Assignment"),
		},
		// Fail TransistionAssignmentStatus
		{
			ExpectedError:                   errors.New("Fail Activating Assignment"),
			TransitionAssignmentStatusError: errors.New("Fail Activating Assignment"),
		},
	}

	// Iterate through each test in the list
	user := "abc"
	account := "123"
	for _, test := range tests {
		// Setup mocks
		mockDB := &mocks.DBer{}
		if test.Create {
			mockDB.On("PutAccountAssignment", mock.Anything).Return(
				test.PutAccountAssignmentAccountAssignment,
				test.PutAccountAssignmentError)
		} else {
			mockDB.On("TransitionAssignmentStatus", mock.Anything,
				mock.Anything, mock.Anything, mock.Anything).Return(
				test.TransitionAssignmentStatusAssignment,
				test.TransitionAssignmentStatusError)
		}

		// Create the AccountProvision
		prov := AccountProvision{
			DBSvc: mockDB,
		}

		// Call findUserAssignmentWithAccount
		assgn, err := prov.ActivateAccountAssignment(test.Create, user, account)

		// Assert that the expected output is correct
		if test.ExpectedAccountAssignment != nil {
			require.Equal(t, test.ExpectedAccountAssignment.AccountID,
				assgn.AccountID)
			require.Equal(t, test.ExpectedAccountAssignment.UserID, assgn.UserID)
			require.Equal(t, test.ExpectedAccountAssignment.AssignmentStatus,
				assgn.AssignmentStatus)
			if test.Create {
				require.NotEqual(t, test.ExpectedAccountAssignment.CreatedOn,
					assgn.CreatedOn) // Should be different
			} else {
				require.Equal(t, test.ExpectedAccountAssignment.CreatedOn,
					assgn.CreatedOn) // Should be the same
			}
			require.NotEqual(t, test.ExpectedAccountAssignment.LastModifiedOn,
				assgn.LastModifiedOn) // Should not be 0
		} else {
			require.Equal(t, test.ExpectedAccountAssignment, assgn)
		}
		require.Equal(t, test.ExpectedError, err)
	}
}

// testRollbackProvisionAccountInput is the structure input used for table
// driven testing for RollbackProvisionAccount
type testRollbackProvisionAccountInput struct {
	ExpectedError                   error
	TransitionAccountStatus         bool
	TransitionAssignmentStatusError error
	TransitionAccountStatusError    error
}

// TestRollbackProvisionAccount tests and verifies the flow of the helper
// function to rollback provisioning an account
func TestRollbackProvisionAccount(t *testing.T) {
	// Construct test scenarios
	tests := []testRollbackProvisionAccountInput{
		// Happy Path - Only Account Assignment revert
		{},
		// Happy Path - Account and Account Assignment revert
		{
			TransitionAccountStatus: true,
		},
		// Fail TransitionAssignmentStatus Only
		{
			ExpectedError:           errors.New("Fail to Revert Assignment"),
			TransitionAccountStatus: true,
			TransitionAssignmentStatusError: errors.New(
				"Fail to Revert Assignment"),
		},
		// Fail TransitionAccountStatus Only
		{
			ExpectedError:                errors.New("Fail to Revert Account"),
			TransitionAccountStatus:      true,
			TransitionAccountStatusError: errors.New("Fail to Revert Account"),
		},
		// Fail Both Reverts
		{
			ExpectedError:           errors.New("Fail to Revert Account"),
			TransitionAccountStatus: true,
			TransitionAssignmentStatusError: errors.New(
				"Fail to Revert Assignment"),
			TransitionAccountStatusError: errors.New("Fail to Revert Account"),
		},
	}

	// Iterate through each test in the list
	user := "abc"
	account := "123"
	for _, test := range tests {
		// Setup mocks
		mockDB := &mocks.DBer{}
		mockDB.On("TransitionAssignmentStatus", mock.Anything,
			mock.Anything, mock.Anything, mock.Anything).Return(
			nil, test.TransitionAssignmentStatusError)
		if test.TransitionAccountStatus {
			mockDB.On("TransitionAccountStatus", mock.Anything,
				mock.Anything, mock.Anything, mock.Anything).Return(
				nil, test.TransitionAccountStatusError)
		}

		// Create the AccountProvision
		prov := AccountProvision{
			DBSvc: mockDB,
		}

		// Call findUserAssignmentWithAccount
		err := prov.RollbackProvisionAccount(test.TransitionAccountStatus, user,
			account)

		// Assert that the expected output is correct
		require.Equal(t, test.ExpectedError, err)
	}
}
