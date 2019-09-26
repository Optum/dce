package provision

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Optum/Dce/pkg/db"
	"github.com/Optum/Dce/pkg/db/mocks"
)

// testFindActiveLeaseForPrincipalInput is the structure input used for table
// driven testing for FindActiveLeaseForPrincipal
type testFindActiveLeaseForPrincipalInput struct {
	ExpectedError             error
	ExpectedLeasePrincipalID  string
	ExpectedLeaseAccountID    string
	FindLeaseByPrincipal      []*db.DceLease
	FindLeaseByPrincipalError error
	ExpectLease               bool
}

// TestFindActiveLeaseFoPrincipal tests and verifies the flow of the helper
// function to find any active leases
func TestFindActiveLeaseFoPrincipal(t *testing.T) {
	// Construct test scenarios
	tests := []testFindActiveLeaseForPrincipalInput{
		// Happy Path - Decommissioned
		{
			ExpectedError:            nil,
			ExpectedLeasePrincipalID: "",
			ExpectedLeaseAccountID:   "",
			FindLeaseByPrincipal: []*db.DceLease{
				&db.DceLease{
					PrincipalID: "abc",
					AccountID:   "123",
					LeaseStatus: db.Decommissioned,
				},
			},
			ExpectLease: true,
		},
		// Happy Path - Active
		{
			ExpectedError:            nil,
			ExpectedLeasePrincipalID: "abc",
			ExpectedLeaseAccountID:   "123",
			FindLeaseByPrincipal: []*db.DceLease{
				&db.DceLease{
					PrincipalID: "abc",
					AccountID:   "123",
					LeaseStatus: db.Active,
				},
			},
			ExpectLease: true,
		},
		// Happy Path - FinanceLock
		{
			ExpectedError:            nil,
			ExpectedLeasePrincipalID: "abc",
			ExpectedLeaseAccountID:   "123",
			FindLeaseByPrincipal: []*db.DceLease{
				&db.DceLease{
					PrincipalID: "abc",
					AccountID:   "123",
					LeaseStatus: db.FinanceLock,
				},
			},
			ExpectLease: true,
		},
		// Happy Path - ResetLock
		{
			ExpectedError:            nil,
			ExpectedLeasePrincipalID: "abc",
			ExpectedLeaseAccountID:   "123",
			FindLeaseByPrincipal: []*db.DceLease{
				&db.DceLease{
					PrincipalID: "abc",
					AccountID:   "123",
					LeaseStatus: db.ResetLock,
				},
			},
			ExpectLease: true,
		},
		// Error FindLeasesByPrincipal
		{
			ExpectedError:             errors.New("Error Finding Lease"),
			FindLeaseByPrincipalError: errors.New("Error Finding Lease"),
		},
	}

	// Iterate through each test in the list
	principalID := "abc"
	for _, test := range tests {
		// Setup mocks
		mockDB := &mocks.DBer{}
		mockDB.On("FindLeasesByPrincipal", mock.Anything).Return(
			test.FindLeaseByPrincipal,
			test.FindLeaseByPrincipalError)

		// Create the AccountProvision
		prov := AccountProvision{
			DBSvc: mockDB,
		}

		// Call FindActiveLeaseForPrincipal
		lease, err := prov.FindActiveLeaseForPrincipal(principalID)

		// Assert that the expected output is correct
		require.Equal(t, test.ExpectedError, err)
		if test.ExpectLease {
			require.Equal(t, test.ExpectedLeasePrincipalID, lease.PrincipalID)
			require.Equal(t, test.ExpectedLeaseAccountID,
				lease.AccountID)
		} else {
			require.Nil(t, lease)
		}
	}
}

// testFindLeaseWithAccountInput is the structure input used for table
// driven testing for FindLeaseWithAccount
type testFindLeaseWithAccountInput struct {
	ExpectedError            error
	ExpectedLeasePrincipalID string
	ExpectedLeaseAccountID   string
	FindLeasesByAccount      []*db.DceLease
	FindLeasesByAccountError error
	ExpectLease              bool
}

// TestFindLeaseWithAccount tests and verifies the flow of the helper
// function to find an lease between a principal and an account
func TestFindLeaseWithAccount(t *testing.T) {
	// Construct test scenarios
	tests := []testFindLeaseWithAccountInput{
		// Happy Path - Lease Exists
		{
			ExpectedError:            nil,
			ExpectedLeasePrincipalID: "abc",
			ExpectedLeaseAccountID:   "123",
			FindLeasesByAccount: []*db.DceLease{
				&db.DceLease{
					PrincipalID: "abc",
					AccountID:   "123",
					LeaseStatus: db.Decommissioned,
				},
			},
			ExpectLease: true,
		},
		// Happy Path - Lease Does Not Exist
		{
			ExpectedError:            nil,
			ExpectedLeasePrincipalID: "",
			ExpectedLeaseAccountID:   "",
			FindLeasesByAccount: []*db.DceLease{
				&db.DceLease{
					PrincipalID: "def",
					AccountID:   "123",
					LeaseStatus: db.Decommissioned,
				},
			},
			ExpectLease: true,
		},
		// Error FindLeasesByAccount
		{
			ExpectedError:            errors.New("Error Finding Lease"),
			FindLeasesByAccountError: errors.New("Error Finding Lease"),
		},
		// Error Account has Active Lease
		{
			ExpectedError: errors.New("Attempt to lease Active Account as " +
				"new Dce - 123"),
			FindLeasesByAccount: []*db.DceLease{
				&db.DceLease{
					PrincipalID: "def",
					AccountID:   "123",
					LeaseStatus: db.Active,
				},
			},
		},
		// Error Account has FinaceLock Lease
		{
			ExpectedError: errors.New("Attempt to lease Active Account as " +
				"new Dce - 123"),
			FindLeasesByAccount: []*db.DceLease{
				&db.DceLease{
					PrincipalID: "def",
					AccountID:   "123",
					LeaseStatus: db.FinanceLock,
				},
			},
		},
		// Error Account has ResetLock Lease
		{
			ExpectedError: errors.New("Attempt to lease Active Account as " +
				"new Dce - 123"),
			FindLeasesByAccount: []*db.DceLease{
				&db.DceLease{
					PrincipalID: "def",
					AccountID:   "123",
					LeaseStatus: db.ResetLock,
				},
			},
		},
	}

	// Iterate through each test in the list
	principalID := "abc"
	accountID := "123"
	for _, test := range tests {
		// Setup mocks
		mockDB := &mocks.DBer{}
		mockDB.On("FindLeasesByAccount", mock.Anything).Return(
			test.FindLeasesByAccount,
			test.FindLeasesByAccountError)

		// Create the AccountProvision
		prov := AccountProvision{
			DBSvc: mockDB,
		}

		// Call findLeaseWithAccount
		lease, err := prov.FindLeaseWithAccount(principalID, accountID)

		// Assert that the expected output is correct
		require.Equal(t, test.ExpectedError, err)
		if test.ExpectLease {
			require.Equal(t, test.ExpectedLeasePrincipalID, lease.PrincipalID)
			require.Equal(t, test.ExpectedLeaseAccountID,
				lease.AccountID)
		} else {
			require.Nil(t, lease)
		}
	}
}

// testActivateLeaseInput is the structure input used for table
// driven testing for ActivateAccount
type testActivateLeaseInput struct {
	ExpectedLease              *db.DceLease
	ExpectedError              error
	Create                     bool
	PutLease                   *db.DceLease
	PutLeaseError              error
	TransitionLeaseStatusLease *db.DceLease
	TransitionLeaseStatusError error
}

// TestActivateLease tests and verifies the flow of the helper
// function to create or update an account lease as active for a principal
func TestActivateLease(t *testing.T) {
	// Construct test scenarios
	lease := &db.DceLease{
		AccountID:   "123",
		PrincipalID: "abc",
		LeaseStatus: db.Active,
	}
	tests := []testActivateLeaseInput{
		// Happy Path - Create
		{
			Create:        true,
			ExpectedLease: lease,
			PutLease:      lease,
		},
		// Happy Path - Update
		{
			ExpectedLease: lease,
			TransitionLeaseStatusLease: &db.DceLease{
				AccountID:             "123",
				PrincipalID:           "abc",
				LeaseStatus:           db.Active,
				LastModifiedOn:        456,
				LeaseStatusModifiedOn: 789,
			},
		},
		// Fail PutLease
		{
			ExpectedError: errors.New("Fail Creating New Lease"),
			Create:        true,
			PutLeaseError: errors.New("Fail Creating New Lease"),
		},
		// Fail TransistionLeaseStatus
		{
			ExpectedError:              errors.New("Fail Activating Lease"),
			TransitionLeaseStatusError: errors.New("Fail Activating Lease"),
		},
	}

	// Iterate through each test in the list
	principalID := "abc"
	accountID := "123"
	var budgetAmount float64 = 300
	budgetCurrency := "USD"
	budgetNotificationEmails := []string{"test@test.com"}

	for _, test := range tests {
		// Setup mocks
		mockDB := &mocks.DBer{}
		if test.Create {
			mockDB.On("PutLease", mock.Anything).Return(
				test.PutLease,
				test.PutLeaseError)
		} else {
			mockDB.On("TransitionLeaseStatus", mock.Anything,
				mock.Anything, mock.Anything, mock.Anything).Return(
				test.TransitionLeaseStatusLease,
				test.TransitionLeaseStatusError)
		}

		// Create the AccountProvision
		prov := AccountProvision{
			DBSvc: mockDB,
		}

		// Call findLeaseWithAccount
		assgn, err := prov.ActivateAccount(test.Create, principalID, accountID, budgetAmount, budgetCurrency, budgetNotificationEmails)

		// Assert that the expected output is correct
		if test.ExpectedLease != nil {
			require.Equal(t, test.ExpectedLease.AccountID,
				assgn.AccountID)
			require.Equal(t, test.ExpectedLease.PrincipalID, assgn.PrincipalID)
			require.Equal(t, test.ExpectedLease.LeaseStatus,
				assgn.LeaseStatus)
			if test.Create {
				require.NotEqual(t, test.ExpectedLease.CreatedOn,
					assgn.CreatedOn) // Should be different
			} else {
				require.Equal(t, test.ExpectedLease.CreatedOn,
					assgn.CreatedOn) // Should be the same
			}
			require.NotEqual(t, test.ExpectedLease.LastModifiedOn,
				assgn.LastModifiedOn) // Should not be 0
			require.NotEqual(t, test.ExpectedLease.LeaseStatusModifiedOn,
				assgn.LeaseStatusModifiedOn) // Should not be 0
		} else {
			require.Equal(t, test.ExpectedLease, assgn)
		}
		require.Equal(t, test.ExpectedError, err)
	}
}

// testRollbackProvisionAccountInput is the structure input used for table
// driven testing for RollbackProvisionAccount
type testRollbackProvisionAccountInput struct {
	ExpectedError                error
	TransitionAccountStatus      bool
	TransitionLeaseStatusError   error
	TransitionAccountStatusError error
}

// TestRollbackProvisionAccount tests and verifies the flow of the helper
// function to rollback provisioning an account
func TestRollbackProvisionAccount(t *testing.T) {
	// Construct test scenarios
	tests := []testRollbackProvisionAccountInput{
		// Happy Path - Only Account Lease revert
		{},
		// Happy Path - Account and Account Lease revert
		{
			TransitionAccountStatus: true,
		},
		// Fail TransitionLeaseStatus Only
		{
			ExpectedError:           errors.New("Fail to Revert Lease"),
			TransitionAccountStatus: true,
			TransitionLeaseStatusError: errors.New(
				"Fail to Revert Lease"),
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
			TransitionLeaseStatusError: errors.New(
				"Fail to Revert Lease"),
			TransitionAccountStatusError: errors.New("Fail to Revert Account"),
		},
	}

	// Iterate through each test in the list
	principalID := "abc"
	accountID := "123"
	for _, test := range tests {
		// Setup mocks
		mockDB := &mocks.DBer{}
		mockDB.On("TransitionLeaseStatus", mock.Anything,
			mock.Anything, mock.Anything, mock.Anything).Return(
			nil, test.TransitionLeaseStatusError)
		if test.TransitionAccountStatus {
			mockDB.On("TransitionAccountStatus", mock.Anything,
				mock.Anything, mock.Anything, mock.Anything).Return(
				nil, test.TransitionAccountStatusError)
		}

		// Create the AccountProvision
		prov := AccountProvision{
			DBSvc: mockDB,
		}

		// Call findLeaseWithAccount
		err := prov.RollbackProvisionAccount(test.TransitionAccountStatus, principalID,
			accountID)

		// Assert that the expected output is correct
		require.Equal(t, test.ExpectedError, err)
	}
}
