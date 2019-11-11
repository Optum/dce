package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDb(t *testing.T) {
	// Load Terraform outputs
	tfOpts := &terraform.Options{
		TerraformDir: "../../modules",
	}
	tfOut := terraform.OutputAll(t, tfOpts)

	// Configure the DB service
	awsSession, err := session.NewSession()
	require.Nil(t, err)
	dbSvc := db.New(
		dynamodb.New(
			awsSession,
			aws.NewConfig().WithRegion(tfOut["aws_region"].(string)),
		),
		tfOut["accounts_table_name"].(string),
		tfOut["leases_table_name"].(string),
		7,
	)
	// Set consistent reads to improve testing without a bunch of sleeps
	// for eventual consistency
	dbSvc.ConsistentRead = true

	// Truncate tables, to make sure we're starting off clean
	truncateDBTables(t, dbSvc)

	t.Run("GetAccount / PutAccount", func(t *testing.T) {
		t.Run("Should retrieve an Account by Id", func(t *testing.T) {
			// Cleanup table on completion
			defer truncateAccountTable(t, dbSvc)

			// Create mock accounts
			accountIds := []string{"111", "222", "333"}
			timeNow := time.Now().Unix()
			for _, acctID := range accountIds {
				a := newAccount(acctID, timeNow)
				a.Metadata = map[string]interface{}{"hello": "world"}
				err := dbSvc.PutAccount(*a)
				require.Nil(t, err)
			}

			// Retrieve the Account, check that it matches our mock
			acct, err := dbSvc.GetAccount("222")
			require.Nil(t, err)
			expected := newAccount("222", timeNow)
			expected.Metadata = map[string]interface{}{"hello": "world"}
			require.Equal(t, expected, acct)
		})

		t.Run("Should return Nil if no account is found", func(t *testing.T) {
			// Try getting an account that doesn't exist
			acct, err := dbSvc.GetAccount("NotAnAccount")
			require.Nil(t, err)
			require.Nil(t, acct)
		})
	})

	t.Run("GetReadyAccount", func(t *testing.T) {
		// Get the first Ready Account
		t.Run("Should retrieve an Account by Ready Status", func(t *testing.T) {
			// Cleanup table on completion
			defer truncateAccountTable(t, dbSvc)

			// Create mock accounts
			timeNow := time.Now().Unix()
			accountNotReady := db.Account{
				ID:            "111",
				AccountStatus: "NotReady",
			}
			err := dbSvc.PutAccount(accountNotReady)
			require.Nil(t, err)
			err = dbSvc.PutAccount(*newAccount("222", timeNow)) // Ready
			require.Nil(t, err)

			// Retrieve the first Ready Account, check that it matches our
			// mock
			acct, err := dbSvc.GetReadyAccount()
			require.Nil(t, err)
			require.Equal(t, newAccount("222", timeNow), acct)
		})

		// Return nil if no Ready Accounts are available
		t.Run("Should return Nil if no account is ready", func(t *testing.T) {
			// Try getting a ready account that doesn't exist
			acct, err := dbSvc.GetReadyAccount()
			require.Nil(t, acct)
			require.Nil(t, err)

			// Cleanup table on completion
			defer truncateAccountTable(t, dbSvc)

			// Create NotReady mock accounts
			accountNotReady := db.Account{
				ID:            "111",
				AccountStatus: "NotReady",
			}
			err = dbSvc.PutAccount(accountNotReady)
			require.Nil(t, err)

			// Verify no account is still ready
			acct, err = dbSvc.GetReadyAccount()
			require.Nil(t, acct)
			require.Nil(t, err)
		})
	})

	t.Run("FindAccountsByStatus", func(t *testing.T) {

		t.Run("should return matching accounts", func(t *testing.T) {
			defer truncateAccountTable(t, dbSvc)

			// Create some accounts in the DB
			for _, acct := range []db.Account{
				{ID: "1", AccountStatus: db.Ready},
				{ID: "2", AccountStatus: db.NotReady},
				{ID: "3", AccountStatus: db.Ready},
				{ID: "4", AccountStatus: db.Leased},
			} {
				err := dbSvc.PutAccount(acct)
				require.Nil(t, err)
			}

			// Find ready accounts
			res, err := dbSvc.FindAccountsByStatus(db.Ready)
			require.Nil(t, err)
			require.Equal(t, []*db.Account{
				{ID: "1", AccountStatus: db.Ready},
				{ID: "3", AccountStatus: db.Ready},
			}, res)
		})

		t.Run("should return an empty list, if none match", func(t *testing.T) {
			defer truncateAccountTable(t, dbSvc)

			// Create some accounts in the DB
			for _, acct := range []db.Account{
				{ID: "1", AccountStatus: db.NotReady},
				{ID: "2", AccountStatus: db.Leased},
			} {
				err := dbSvc.PutAccount(acct)
				require.Nil(t, err)
			}

			// Find ready accounts
			res, err := dbSvc.FindAccountsByStatus(db.Ready)
			require.Nil(t, err)
			require.Equal(t, []*db.Account{}, res)
		})

	})

	t.Run("TransitionAccountStatus", func(t *testing.T) {
		require.NotNil(t, "TODO")
	})

	t.Run("UpdateIamPolicyHash", func(t *testing.T) {
		t.Run("Create IAM Policy Hash from blank and still get account", func(t *testing.T) {
			require.NotNil(t, "TODO")
		})

		t.Run("Create IAM Policy Hash from value and still get account", func(t *testing.T) {
			defer truncateLeaseTable(t, dbSvc)

			// Create some accounts in the DB
			for _, acct := range []db.Account{
				{ID: "1", AccountStatus: db.Ready, PrincipalPolicyHash: "\"PreviousHash\""},
				{ID: "2", AccountStatus: db.Leased},
			} {
				err := dbSvc.PutAccount(acct)
				require.Nil(t, err)
			}

			// Find ready accounts
			res, err := dbSvc.UpdateAccountPrincipalPolicyHash("1", "\"PreviousHash\"", "\"NextHash\"")
			require.Nil(t, err)
			require.Equal(t, res.PrincipalPolicyHash, "\"NextHash\"")

			res, err = dbSvc.GetAccount("1")
			require.Nil(t, err)
		})
	})

	t.Run("TransitionLeaseStatus", func(t *testing.T) {

		t.Run("Should transition from one state to another", func(t *testing.T) {
			// Cleanup DB when we're done
			defer truncateLeaseTable(t, dbSvc)

			// Create a mock lease with Status=Active
			acctID := "111"
			principalID := "222"
			timeNow := time.Now().Unix()
			lease := db.Lease{
				ID:                    uuid.New().String(),
				AccountID:             acctID,
				PrincipalID:           principalID,
				LeaseStatus:           db.Active,
				LeaseStatusReason:     db.LeaseActive,
				CreatedOn:             timeNow,
				LastModifiedOn:        timeNow,
				LeaseStatusModifiedOn: timeNow,
			}
			putAssgn, err := dbSvc.PutLease(lease)
			// Check for the error first, because PutLease will return a nil lease upon error
			// and this test is easier to catch and diagnose if it fails.
			require.Nil(t, err, "Expected no errors saving a new lease to the db.")
			require.Equal(t, db.Lease{}, *putAssgn) // should return an empty account lease since its new
			leaseBefore, err := dbSvc.GetLease(acctID, principalID)

			time.Sleep(1 * time.Second) // Ensure LastModifiedOn and LeaseStatusModifiedOn changes
			// Set a ResetLock on the Lease
			updatedLease, err := dbSvc.TransitionLeaseStatus(
				acctID, principalID,
				db.Active, db.Inactive,
				db.LeaseDestroyed,
			)
			require.Nil(t, err)
			require.NotNil(t, updatedLease)

			// Check that the returned Lease
			// has Status=ResetLock
			require.Equal(t, updatedLease.LeaseStatus, db.Inactive)

			// Check the lease in the db
			leaseAfter, err := dbSvc.GetLease(acctID, principalID)
			require.Nil(t, err)
			require.NotNil(t, leaseAfter)
			require.Equal(t, leaseAfter.LeaseStatus, db.Inactive)
			require.True(t, leaseBefore.LastModifiedOn !=
				leaseAfter.LastModifiedOn)
			require.True(t, leaseBefore.LeaseStatusModifiedOn !=
				leaseAfter.LeaseStatusModifiedOn)
		})

		t.Run("Should fail if the Lease does not exit", func(t *testing.T) {
			// Attempt to lock an lease that doesn't exist
			updatedLease, err := dbSvc.TransitionLeaseStatus(
				"not-an-acct-id", "not-a-principal-id",
				db.Active, db.Inactive,
				db.LeaseDestroyed,
			)
			require.NotNil(t, err)
			require.Nil(t, updatedLease)

			assert.Equal(t, "unable to update lease status from \"Active\" to \"Inactive\" for not-an-acct-id/not-a-principal-id: "+
				"no lease exists with Status=\"Active\"", err.Error())
		})

		t.Run("Should fail if account is not in prevStatus", func(t *testing.T) {
			// Run test for each non-active status
			notActiveStatuses := []db.LeaseStatus{db.Inactive}
			for _, status := range notActiveStatuses {

				t.Run(fmt.Sprint("...when status is ", status), func(t *testing.T) {
					// Cleanup DB when we're done
					defer truncateLeaseTable(t, dbSvc)

					// Create a mock lease
					// with our non-active status
					acctID := "111"
					principalID := "222"
					timeNow := time.Now().Unix()
					lease := db.Lease{
						ID:                uuid.New().String(),
						AccountID:         acctID,
						PrincipalID:       principalID,
						LeaseStatus:       status,
						LeaseStatusReason: db.LeaseActive,
						CreatedOn:         timeNow,
						LastModifiedOn:    timeNow,
					}
					putAssgn, err := dbSvc.PutLease(lease)
					require.Nil(t, err, "Expected no errors saving a new lease to the db.")
					require.Equal(t, db.Lease{}, *putAssgn) // should return an empty account lease since its new

					// Attempt to set a ResetLock on the Lease
					updatedLease, err := dbSvc.TransitionLeaseStatus(
						acctID, principalID,
						db.Active, status,
						db.LeaseExpired,
					)
					require.NotNil(t, err)
					require.Nil(t, updatedLease)

					require.IsType(t, &db.StatusTransitionError{}, err)
					assert.Equal(t, fmt.Sprintf("unable to update lease status from \"Active\" to \"%v\" for 111/222: "+
						"no lease exists with Status=\"Active\"", status), err.Error())
				})

			}
		})

	})

	t.Run("TransitionAccountStatus", func(t *testing.T) {

		t.Run("Should transition from one state to another", func(t *testing.T) {
			// Cleanup DB when we're done
			defer truncateAccountTable(t, dbSvc)

			// Create a mock lease with Status=Active
			acctID := "111"
			timeNow := time.Now().Unix()
			account := db.Account{
				ID:             acctID,
				AccountStatus:  db.Leased,
				LastModifiedOn: timeNow,
			}
			err := dbSvc.PutAccount(account)
			require.Nil(t, err)
			accountBefore, err := dbSvc.GetAccount(acctID)

			time.Sleep(1 * time.Second) // Ensure LastModifiedOn and LeaseStatusModifiedOn changes
			// Set a ResetLock on the Lease
			updatedAccount, err := dbSvc.TransitionAccountStatus(
				acctID,
				db.Leased, db.Ready,
			)
			require.Nil(t, err)
			require.NotNil(t, updatedAccount)

			// Check that the returned account
			// has Status=Ready
			require.Equal(t, updatedAccount.AccountStatus, db.Ready)

			// Check the account in the db got updated
			accountAfter, err := dbSvc.GetAccount(acctID)
			require.Nil(t, err)
			require.NotNil(t, accountAfter)
			require.Equal(t, accountAfter.AccountStatus, db.Ready)
			require.True(t, accountBefore.LastModifiedOn !=
				accountAfter.LastModifiedOn)
		})

		t.Run("Should fail if the Account does not exit", func(t *testing.T) {
			// Attempt to modify an account that doesn't exist
			updatedAccount, err := dbSvc.TransitionAccountStatus(
				"not-an-acct-id",
				db.NotReady, db.Ready,
			)
			require.Nil(t, updatedAccount)
			require.NotNil(t, err)

			assert.Equal(t, "unable to update account status from \"NotReady\" to \"Ready\" for account not-an-acct-id: "+
				"no account exists with Status=\"NotReady\"", err.Error())
		})

		t.Run("Should fail if account is not in prevStatus", func(t *testing.T) {
			// Run test for each status except "Ready"
			notActiveStatuses := []db.AccountStatus{db.NotReady, db.Leased}
			for _, status := range notActiveStatuses {

				t.Run(fmt.Sprint("...when status is ", status), func(t *testing.T) {
					// Cleanup DB when we're done
					defer truncateAccountTable(t, dbSvc)

					// Create a mock account
					// with our non-active status
					acctID := "111"
					account := db.Account{
						ID:            acctID,
						AccountStatus: status,
					}
					err := dbSvc.PutAccount(account)
					require.Nil(t, err)

					// Attempt to change status from Ready -> NotReady
					// (should fail, because the account is not currently
					updatedAccount, err := dbSvc.TransitionAccountStatus(
						acctID,
						db.Ready, db.NotReady,
					)
					require.Nil(t, updatedAccount)
					require.NotNil(t, err)

					require.IsType(t, &db.StatusTransitionError{}, err)
					require.Equal(t, "unable to update account status from \"Ready\" to \"NotReady\" for account 111: "+
						"no account exists with Status=\"Ready\"", err.Error())
				})

			}
		})

	})

	t.Run("FindLeasesByAccount", func(t *testing.T) {

		t.Run("Find Existing Account", func(t *testing.T) {
			// Cleanup DB when we're done
			defer truncateLeaseTable(t, dbSvc)

			// Create a mock lease
			// with our non-active status
			acctID := "111"
			principalID := "222"
			status := db.Active
			timeNow := time.Now().Unix()
			lease := db.Lease{
				ID:                    uuid.New().String(),
				AccountID:             acctID,
				PrincipalID:           principalID,
				LeaseStatus:           status,
				LeaseStatusReason:     db.LeaseActive,
				CreatedOn:             timeNow,
				LastModifiedOn:        timeNow,
				LeaseStatusModifiedOn: timeNow,
			}
			putAssgn, err := dbSvc.PutLease(lease)
			require.Nil(t, err, "Expected no errors saving a new lease to the db.")
			require.Equal(t, db.Lease{}, *putAssgn) // should return an empty account lease since its new

			foundaccount, err := dbSvc.FindLeasesByAccount("111")

			require.NotNil(t, foundaccount)
			require.Nil(t, err)
		})

		t.Run("Fail to find non-existent Account", func(t *testing.T) {
			// Cleanup DB when we're done
			defer truncateLeaseTable(t, dbSvc)

			// Create a mock lease
			// with our non-active status
			acctID := "333"
			principalID := "222"
			status := db.Active
			timeNow := time.Now().Unix()
			lease := db.Lease{
				ID:                    uuid.New().String(),
				AccountID:             acctID,
				PrincipalID:           principalID,
				LeaseStatus:           status,
				LeaseStatusReason:     db.LeaseActive,
				CreatedOn:             timeNow,
				LastModifiedOn:        timeNow,
				LeaseStatusModifiedOn: timeNow,
			}
			putAssgn, err := dbSvc.PutLease(lease)
			require.Equal(t, db.Lease{}, *putAssgn) // should return an empty account lease since its new
			require.Nil(t, err)

			foundLease, err := dbSvc.FindLeasesByAccount("111")

			// require.Nil(t, foundLease)
			require.Empty(t, foundLease)
			require.Nil(t, err)
		})
	})

	t.Run("FindLeasesByPrincipal", func(t *testing.T) {

		t.Run("Find Existing Principal", func(t *testing.T) {
			// Cleanup DB when we're done
			defer truncateLeaseTable(t, dbSvc)

			// Create a mock lease
			// with our non-active status
			acctID := "111"
			principalID := "222"
			status := db.Active
			lease := db.Lease{
				ID:                uuid.New().String(),
				AccountID:         acctID,
				PrincipalID:       principalID,
				LeaseStatus:       status,
				LeaseStatusReason: db.LeaseActive,
			}
			putAssgn, err := dbSvc.PutLease(lease)
			require.Equal(t, db.Lease{}, *putAssgn) // should return an empty account lease since its new

			foundaccount, err := dbSvc.FindLeasesByPrincipal("222")

			require.NotNil(t, foundaccount)
			require.Nil(t, err)
		})

		t.Run("Fail to find non-existent Lease", func(t *testing.T) {
			// Cleanup DB when we're done
			defer truncateLeaseTable(t, dbSvc)

			// Create a mock lease
			// with our non-active status
			acctID := "333"
			principalID := "222"
			status := db.Active
			lease := db.Lease{
				ID:                uuid.New().String(),
				AccountID:         acctID,
				PrincipalID:       principalID,
				LeaseStatus:       status,
				LeaseStatusReason: db.LeaseActive,
			}
			putAssgn, err := dbSvc.PutLease(lease)
			require.Equal(t, db.Lease{}, *putAssgn) // should return an empty account lease since its new
			require.Nil(t, err)

			foundLease, err := dbSvc.FindLeasesByPrincipal("111")

			require.Nil(t, foundLease)
			require.Nil(t, err)
		})
	})

	t.Run("FindLeasesByStatus", func(t *testing.T) {

		t.Run("should return leases matching a status", func(t *testing.T) {
			defer truncateLeaseTable(t, dbSvc)

			// Make up the IDs ahead of time, because the assertinons later will want them...
			uuidOne := uuid.New().String()
			uuidTwo := uuid.New().String()
			uuidThree := uuid.New().String()
			uuidFour := uuid.New().String()

			expiryDate := time.Now().AddDate(0, 0, 30).Unix()

			// Create some leases in the DB
			for _, lease := range []db.Lease{
				{ID: uuidOne, AccountID: "1", PrincipalID: "pid", LeaseStatus: db.Active, LeaseStatusReason: db.LeaseActive, ExpiresOn: expiryDate},
				{ID: uuidTwo, AccountID: "2", PrincipalID: "pid", LeaseStatus: db.Inactive, LeaseStatusReason: db.LeaseExpired, ExpiresOn: expiryDate},
				{ID: uuidThree, AccountID: "3", PrincipalID: "pid", LeaseStatus: db.Active, LeaseStatusReason: db.LeaseActive, ExpiresOn: expiryDate},
				{ID: uuidFour, AccountID: "4", PrincipalID: "pid", LeaseStatus: db.Inactive, LeaseStatusReason: db.LeaseDestroyed, ExpiresOn: expiryDate},
			} {
				_, err := dbSvc.PutLease(lease)
				require.Nil(t, err)
			}

			// Find ResetLock leases
			res, err := dbSvc.FindLeasesByStatus(db.Inactive)
			require.Nil(t, err)
			require.Equal(t, []*db.Lease{
				{ID: uuidTwo, AccountID: "2", PrincipalID: "pid", LeaseStatus: db.Inactive, LeaseStatusReason: db.LeaseExpired, ExpiresOn: expiryDate},
				{ID: uuidFour, AccountID: "4", PrincipalID: "pid", LeaseStatus: db.Inactive, LeaseStatusReason: db.LeaseDestroyed, ExpiresOn: expiryDate},
			}, res)
		})

		t.Run("should return an empty list if none match", func(t *testing.T) {
			defer truncateLeaseTable(t, dbSvc)

			// Create some leases in the DB
			for _, lease := range []db.Lease{
				{ID: uuid.New().String(), AccountID: "1", PrincipalID: "pid", LeaseStatus: db.Active, LeaseStatusReason: db.LeaseActive},
				{ID: uuid.New().String(), AccountID: "2", PrincipalID: "pid", LeaseStatus: db.Active, LeaseStatusReason: db.LeaseActive},
			} {
				_, err := dbSvc.PutLease(lease)
				require.Nil(t, err)
			}

			// Find ResetLock leases
			res, err := dbSvc.FindLeasesByStatus(db.Inactive)
			require.Nil(t, err)
			require.Equal(t, []*db.Lease{}, res)
		})

	})

	t.Run("GetAccounts", func(t *testing.T) {
		t.Run("returns a list of accounts", func(t *testing.T) {
			defer truncateAccountTable(t, dbSvc)
			expectedID := "1234123412"
			account := *newAccount(expectedID, 1561382309)
			err := dbSvc.PutAccount(account)
			require.Nil(t, err)

			accounts, err := dbSvc.GetAccounts()
			require.Nil(t, err)
			require.True(t, true, len(accounts) > 0)
			require.Equal(t, accounts[0].ID, expectedID, "The ID of the returns record should match the expected ID")
		})
	})

	t.Run("DeleteAccount", func(t *testing.T) {
		accountID := "1234123412"

		t.Run("when the account exists", func(t *testing.T) {
			t.Run("when the account is not leased", func(t *testing.T) {
				defer truncateAccountTable(t, dbSvc)
				account := *newAccount(accountID, 1561382309)
				err := dbSvc.PutAccount(account)
				require.Nil(t, err, "it returns no errors")
				returnedAccount, err := dbSvc.DeleteAccount(accountID)
				require.Equal(t, account.ID, returnedAccount.ID, "returned account matches the deleted account")
				require.Nil(t, err, "it returns no errors on delete")
				deletedAccount, err := dbSvc.GetAccount(accountID)
				require.Nil(t, deletedAccount, "the account is deleted")
				require.Nil(t, err, "it returns no errors")
			})

			t.Run("when the account is leased", func(t *testing.T) {
				defer truncateAccountTable(t, dbSvc)
				account := db.Account{
					ID:             accountID,
					AccountStatus:  db.Leased,
					LastModifiedOn: 1561382309,
				}
				err := dbSvc.PutAccount(account)
				require.Nil(t, err, "it should not error on delete")
				returnedAccount, err := dbSvc.DeleteAccount(accountID)
				require.Equal(t, account.ID, returnedAccount.ID, "returned account matches the deleted account")
				expectedErrorMessage := fmt.Sprintf("Unable to delete account \"%s\": account is leased.", accountID)
				require.NotNil(t, err, "it returns an error")
				assert.IsType(t, &db.AccountLeasedError{}, err)
				require.EqualError(t, err, expectedErrorMessage, "it has the correct error message")
			})
		})

		t.Run("when the account does not exists", func(t *testing.T) {
			nonexistentAccount, err := dbSvc.DeleteAccount(accountID)
			require.Nil(t, nonexistentAccount, "no account is returned")
			require.NotNil(t, err, "it returns an error")
			expectedErrorMessage := fmt.Sprintf("No account found with ID \"%s\".", accountID)
			require.EqualError(t, err, expectedErrorMessage, "it has the correct error message")
			assert.IsType(t, &db.AccountNotFoundError{}, err)
		})
	})

	t.Run("UpdateMetadata", func(t *testing.T) {
		defer truncateAccountTable(t, dbSvc)
		id := "test-metadata"
		account := db.Account{ID: id, AccountStatus: db.Ready}
		err := dbSvc.PutAccount(account)
		require.Nil(t, err)

		expected := map[string]interface{}{
			"sso": map[string]interface{}{
				"hello": "world",
			},
		}

		err = dbSvc.UpdateMetadata(id, expected)
		require.Nil(t, err)

		updatedAccount, err := dbSvc.GetAccount(id)
		require.Equal(t, expected, updatedAccount.Metadata, "Metadata should be updated")
		require.NotEqual(t, 0, updatedAccount.LastModifiedOn, "Last modified is updated")
	})

	t.Run("GetLeases", func(t *testing.T) {
		defer truncateLeaseTable(t, dbSvc)

		accountIDOne := "1"
		accountIDTwo := "2"
		principalIDOne := "a"
		principalIDTwo := "b"
		principalIDThree := "c"
		principalIDFour := "d"

		_, err = dbSvc.PutLease(db.Lease{
			ID:                uuid.New().String(),
			AccountID:         accountIDOne,
			PrincipalID:       principalIDOne,
			LeaseStatus:       db.Active,
			LeaseStatusReason: db.LeaseActive,
		})

		assert.Nil(t, err)

		_, err = dbSvc.PutLease(db.Lease{
			ID:                uuid.New().String(),
			AccountID:         accountIDOne,
			PrincipalID:       principalIDTwo,
			LeaseStatus:       db.Active,
			LeaseStatusReason: db.LeaseActive,
		})

		assert.Nil(t, err)

		_, err = dbSvc.PutLease(db.Lease{
			ID:                uuid.New().String(),
			AccountID:         accountIDOne,
			PrincipalID:       principalIDThree,
			LeaseStatus:       db.Inactive,
			LeaseStatusReason: db.LeaseDestroyed,
		})

		assert.Nil(t, err)

		_, err = dbSvc.PutLease(db.Lease{
			ID:                uuid.New().String(),
			AccountID:         accountIDTwo,
			PrincipalID:       principalIDFour,
			LeaseStatus:       db.Active,
			LeaseStatusReason: db.LeaseActive,
		})

		assert.Nil(t, err)

		_, err = dbSvc.PutLease(db.Lease{
			ID:                uuid.New().String(),
			AccountID:         accountIDTwo,
			PrincipalID:       principalIDOne,
			LeaseStatus:       db.Inactive,
			LeaseStatusReason: db.LeaseDestroyed,
		})

		assert.Nil(t, err)

		t.Run("When there no filters", func(t *testing.T) {
			output, err := dbSvc.GetLeases(db.GetLeasesInput{})
			assert.Nil(t, err)
			assert.Equal(t, 5, len(output.Results), "only two leases should be returned")
		})

		t.Run("When there is a limit", func(t *testing.T) {
			output, err := dbSvc.GetLeases(db.GetLeasesInput{
				Limit: 2,
			})
			assert.Nil(t, err)
			assert.Equal(t, 2, len(output.Results), "only two leases should be returned")
		})

		t.Run("When there is a status", func(t *testing.T) {
			output, err := dbSvc.GetLeases(db.GetLeasesInput{
				Status: string(db.Inactive),
			})
			assert.Nil(t, err)
			assert.Equal(t, 2, len(output.Results), "only one lease should be returned")
			assert.Equal(t, db.Inactive, output.Results[0].LeaseStatus, "lease should be decommissioned")
		})

		t.Run("When there is a principal ID", func(t *testing.T) {
			output, err := dbSvc.GetLeases(db.GetLeasesInput{
				PrincipalID: principalIDOne,
			})
			assert.Nil(t, err)
			assert.Equal(t, len(output.Results), 2, "should only return one lease")
			assert.Equal(t, output.Results[0].PrincipalID, principalIDOne, "should return the lease with the given ID")
		})

		t.Run("When there is an account ID", func(t *testing.T) {
			output, err := dbSvc.GetLeases(db.GetLeasesInput{
				AccountID: accountIDTwo,
			})
			assert.Nil(t, err)
			assert.Equal(t, 2, len(output.Results), "only one lease should be returned")
		})

		t.Run("When there is a start key", func(t *testing.T) {
			results := make([]*db.Lease, 0)

			shouldContinue := true
			next := make(map[string]string)

			for shouldContinue {
				output, err := dbSvc.GetLeases(db.GetLeasesInput{
					Limit:     2,
					Status:    string(db.Active),
					StartKeys: next,
				})

				assert.Nil(t, err)
				next = output.NextKeys
				results = append(results, output.Results...)

				if len(next) == 0 {
					shouldContinue = false
				}
			}

			assert.Equal(t, 3, len(results), "only three leases should be returned")
			assert.Equal(t, results[0].LeaseStatus, db.Active)
			assert.Equal(t, results[1].LeaseStatus, db.Active)
			assert.Equal(t, results[2].LeaseStatus, db.Active)
		})

		t.Run("When there is an account ID, principal ID, and a lease status", func(t *testing.T) {
			output, err := dbSvc.GetLeases(db.GetLeasesInput{
				AccountID:   accountIDOne,
				PrincipalID: principalIDThree,
				Status:      string(db.Inactive),
			})

			assert.Nil(t, err)
			assert.Equal(t, 1, len(output.Results), "only one lease should be returned")
			assert.Equal(t, db.Inactive, output.Results[0].LeaseStatus, "lease should be decommissioned")
		})
	})

	t.Run("UpsertLease", func(t *testing.T) {
		defer truncateLeaseTable(t, dbSvc)

		t.Run("should create a new lease", func(t *testing.T) {
			truncateLeaseTable(t, dbSvc)

			leaseToCreate := db.Lease{
				AccountID:                "123456789012",
				PrincipalID:              "jdoe123",
				ID:                       "uuid-1234",
				LeaseStatus:              db.Active,
				LeaseStatusReason:        db.LeaseActive,
				CreatedOn:                100,
				LastModifiedOn:           200,
				LeaseStatusModifiedOn:    300,
				ExpiresOn:                400,
				BudgetAmount:             500,
				BudgetCurrency:           "USD",
				BudgetNotificationEmails: []string{"jdoe@example.com"},
				Metadata: map[string]interface{}{
					"foo": "bar",
				},
			}
			leaseRes, err := dbSvc.UpsertLease(leaseToCreate)
			require.Nil(t, err)

			require.Equal(t, &leaseToCreate, leaseRes, "should return the updated leaseToCreate")

			// Lookup the lease in the DB
			foundLease, err := dbSvc.GetLeaseByID("uuid-1234")
			require.Nil(t, err)
			require.Equal(t, &leaseToCreate, foundLease, "Should find the created lease")

			t.Run("should update an existing leaseToCreate", func(t *testing.T) {
				// Make some modifications to our lease object
				leaseToUpdate := leaseToCreate
				leaseToUpdate.LeaseStatus = db.Inactive
				leaseToUpdate.LeaseStatusReason = db.LeaseExpired
				leaseToUpdate.BudgetAmount = 1000
				leaseToUpdate.Metadata = map[string]interface{}{
					"foo": map[string]interface{}{
						"bar": "baz",
					},
				}

				// Update the Lease in the DB
				leaseRes, err = dbSvc.UpsertLease(leaseToUpdate)
				require.Nil(t, err)

				require.Equal(t, &leaseToUpdate, leaseRes, "Should return updated lease")

				// Lookup the updated lease in the DB
				// (note this would fail if a second lease was created
				//  with the same ID as the first)
				foundLease, err = dbSvc.GetLeaseByID("uuid-1234")
				require.Nil(t, err)
				require.Equal(t, &leaseToUpdate, foundLease, "Should return updated lease")
			})
		})

	})
}

func newAccount(id string, timeNow int64) *db.Account {
	account := db.Account{
		ID:             id,
		AccountStatus:  "Ready",
		LastModifiedOn: timeNow,
	}
	return &account
}

// Remove all records from the Account table
func truncateAccountTable(t *testing.T, dbSvc *db.DB) {
	/*
		DynamoDB does not provide a "truncate" method.
		Instead, we need to find all records in the DB table,
		and remove them in a "BatchWrite" requests.
	*/

	// Find all records in the Account table
	scanResult, err := dbSvc.Client.Scan(
		&dynamodb.ScanInput{
			TableName:      aws.String(dbSvc.AccountTableName),
			ConsistentRead: aws.Bool(true),
		},
	)
	require.Nil(t, err)

	if len(scanResult.Items) < 1 {
		return
	}

	// Populate a list of `DeleteRequests` for each item we found in the table
	var deleteRequests []*dynamodb.WriteRequest
	for _, item := range scanResult.Items {
		deleteRequests = append(deleteRequests, &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: map[string]*dynamodb.AttributeValue{
					"Id": item["Id"],
				},
			},
		})
	}

	// Execute Batch requests, to remove all items
	_, err = dbSvc.Client.BatchWriteItem(
		&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				dbSvc.AccountTableName: deleteRequests,
			},
		},
	)
	require.Nil(t, err)
	time.Sleep(2 * time.Second)
}

/*
Remove all records from the Lease table
*/
func truncateLeaseTable(t *testing.T, dbSvc *db.DB) {
	/*
		DynamoDb does not provide a "truncate" method.
		Instead, we need to find all records in the DB table,
		and remove them in a "BatchWrite" requests.
	*/

	// Find all records in the Account table
	scanResult, err := dbSvc.Client.Scan(
		&dynamodb.ScanInput{
			TableName:      aws.String(dbSvc.LeaseTableName),
			ConsistentRead: aws.Bool(true),
		},
	)
	require.Nil(t, err)

	if len(scanResult.Items) < 1 {
		return
	}

	// Populate a list of `DeleteRequests` for each
	// item we found in the table
	var deleteRequests []*dynamodb.WriteRequest
	for _, item := range scanResult.Items {
		deleteRequests = append(deleteRequests, &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: map[string]*dynamodb.AttributeValue{
					"AccountId":   item["AccountId"],
					"PrincipalId": item["PrincipalId"],
				},
			},
		})
	}

	// Execute Batch requests, to remove all items
	_, err = dbSvc.Client.BatchWriteItem(
		&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				dbSvc.LeaseTableName: deleteRequests,
			},
		},
	)
	require.Nil(t, err)
	time.Sleep(2 * time.Second)
}

func truncateDBTables(t *testing.T, dbSvc *db.DB) {
	truncateAccountTable(t, dbSvc)
	truncateLeaseTable(t, dbSvc)
}
