package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/Optum/Redbox/pkg/db"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
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
		tfOut["dynamodb_table_account_name"].(string),
		tfOut["dynamodb_table_account_assignment_name"].(string),
	)

	t.Run("GetAccount / PutAccount", func(t *testing.T) {
		t.Run("Should retrieve an Account by Id", func(t *testing.T) {
			// Cleanup table on completion
			defer truncateAccountTable(t, dbSvc)

			// Create mock accounts
			accountIds := []string{"111", "222", "333"}
			timeNow := time.Now().Unix()
			for _, acctID := range accountIds {
				err := dbSvc.PutAccount(*newAccount(acctID, timeNow))
				require.Nil(t, err)
			}

			// Retrieve the RedboxAccount, check that it matches our mock
			acct, err := dbSvc.GetAccount("222")
			require.Nil(t, err)
			require.Equal(t, newAccount("222", timeNow), acct)
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
			accountNotReady := db.RedboxAccount{
				ID:            "111",
				AccountStatus: "NotReady",
			}
			err := dbSvc.PutAccount(accountNotReady)
			require.Nil(t, err)
			err = dbSvc.PutAccount(*newAccount("222", timeNow)) // Ready
			require.Nil(t, err)

			// Retrieve the first Ready RedboxAccount, check that it matches our
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
			accountNotReady := db.RedboxAccount{
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

	t.Run("GetAccountsForReset", func(t *testing.T) {
		// Get the first Ready Account
		t.Run("Should retrieve Accounts by non-Ready Status", func(t *testing.T) {
			// Cleanup table on completion
			defer truncateAccountTable(t, dbSvc)

			// Create mock accounts
			timeNow := time.Now().Unix()
			err := dbSvc.PutAccount(*newAccount("222", timeNow)) // Ready
			require.Nil(t, err)
			accountNotReady := db.RedboxAccount{
				ID:             "222",
				AccountStatus:  "NotReady",
				LastModifiedOn: timeNow,
			}
			err = dbSvc.PutAccount(accountNotReady)
			require.Nil(t, err)
			accountAssigned := db.RedboxAccount{
				ID:             "333",
				AccountStatus:  "Assigned",
				LastModifiedOn: timeNow,
			}
			err = dbSvc.PutAccount(accountAssigned)
			require.Nil(t, err)

			// Retrieve all RedboxAccount that can be Reset (non-Ready)
			accts, err := dbSvc.GetAccountsForReset()
			require.Nil(t, err)
			require.Equal(t, 2, len(accts))
			require.Equal(t, accountNotReady, *accts[0])
			require.Equal(t, accountAssigned, *accts[1])
		})
	})

	t.Run("TransitionAccountStatus", func(t *testing.T) {
		require.NotNil(t, "TODO")
	})

	t.Run("TransitionAssignmentStatus", func(t *testing.T) {

		t.Run("Should transition from one state to another", func(t *testing.T) {
			// Cleanup DB when we're done
			defer truncateAccountAssignmentTable(t, dbSvc)

			// Create a mock assignment with Status=Active
			acctID := "111"
			userID := "222"
			timeNow := time.Now().Unix()
			assignment := db.RedboxAccountAssignment{
				AccountID:        acctID,
				UserID:           userID,
				AssignmentStatus: db.Active,
				CreatedOn:        timeNow,
				LastModifiedOn:   timeNow,
			}
			putAssgn, err := dbSvc.PutAccountAssignment(assignment)
			require.Equal(t, db.RedboxAccountAssignment{}, *putAssgn) // should return an empty account assignment since its new
			require.Nil(t, err)
			assignmentBefore, err := dbSvc.GetAssignment(acctID, userID)
			time.Sleep(1 * time.Second) // Ensure LastModifiedOn changes

			// Set a ResetLock on the Assignment
			updatedAssignment, err := dbSvc.TransitionAssignmentStatus(
				acctID, userID,
				db.Active, db.ResetLock,
			)
			require.Nil(t, err)
			require.NotNil(t, updatedAssignment)

			// Check that the returned Assignment
			// has Status=ResetLock
			require.Equal(t, updatedAssignment.AssignmentStatus, db.ResetLock)

			// Check the assignment in the db
			assignmentAfter, err := dbSvc.GetAssignment(acctID, userID)
			require.Nil(t, err)
			require.NotNil(t, assignmentAfter)
			require.Equal(t, assignmentAfter.AssignmentStatus, db.ResetLock)
			require.True(t, assignmentBefore.LastModifiedOn !=
				assignmentAfter.LastModifiedOn)
		})

		t.Run("Should fail if the Assignment does not exit", func(t *testing.T) {
			// Attempt to lock an assignment that doesn't exist
			updatedAssignment, err := dbSvc.TransitionAssignmentStatus(
				"not-an-acct-id", "not-a-user-id",
				db.Active, db.ResetLock,
			)
			require.Nil(t, updatedAssignment)
			require.NotNil(t, err)

			assert.Equal(t, "unable to update assignment status from \"Active\" to \"ResetLock\" for not-an-acct-id/not-a-user-id: "+
				"no assignment exists with Status=\"Active\"", err.Error())
		})

		t.Run("Should fail if account is not in prevStatus", func(t *testing.T) {
			// Run test for each non-active status
			notActiveStatuses := []db.AssignmentStatus{db.FinanceLock, db.Decommissioned}
			for _, status := range notActiveStatuses {

				t.Run(fmt.Sprint("...when status is ", status), func(t *testing.T) {
					// Cleanup DB when we're done
					defer truncateAccountAssignmentTable(t, dbSvc)

					// Create a mock assignment
					// with our non-active status
					acctID := "111"
					userID := "222"
					timeNow := time.Now().Unix()
					assignment := db.RedboxAccountAssignment{
						AccountID:        acctID,
						UserID:           userID,
						AssignmentStatus: status,
						CreatedOn:        timeNow,
						LastModifiedOn:   timeNow,
					}
					putAssgn, err := dbSvc.PutAccountAssignment(assignment)
					require.Equal(t, db.RedboxAccountAssignment{}, *putAssgn) // should return an empty account assignment since its new
					require.Nil(t, err)

					// Attempt to set a ResetLock on the Assignment
					updatedAssignment, err := dbSvc.TransitionAssignmentStatus(
						acctID, userID,
						db.Active, status,
					)
					require.Nil(t, updatedAssignment)
					require.NotNil(t, err)

					require.IsType(t, &db.StatusTransitionError{}, err)
					assert.Equal(t, fmt.Sprintf("unable to update assignment status from \"Active\" to \"%v\" for 111/222: "+
						"no assignment exists with Status=\"Active\"", status), err.Error())
				})

			}
		})

	})

	t.Run("TransitionAccountStatus", func(t *testing.T) {

		t.Run("Should transition from one state to another", func(t *testing.T) {
			// Cleanup DB when we're done
			defer truncateAccountTable(t, dbSvc)

			// Create a mock assignment with Status=Active
			acctID := "111"
			timeNow := time.Now().Unix()
			account := db.RedboxAccount{
				ID:             acctID,
				AccountStatus:  db.Assigned,
				LastModifiedOn: timeNow,
			}
			err := dbSvc.PutAccount(account)
			require.Nil(t, err)
			accountBefore, err := dbSvc.GetAccount(acctID)
			time.Sleep(1 * time.Second) // Ensure LastModifiedOn changes

			// Set a ResetLock on the Assignment
			updatedAccount, err := dbSvc.TransitionAccountStatus(
				acctID,
				db.Assigned, db.Ready,
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
			notActiveStatuses := []db.AccountStatus{db.NotReady, db.Assigned}
			for _, status := range notActiveStatuses {

				t.Run(fmt.Sprint("...when status is ", status), func(t *testing.T) {
					// Cleanup DB when we're done
					defer truncateAccountTable(t, dbSvc)

					// Create a mock account
					// with our non-active status
					acctID := "111"
					account := db.RedboxAccount{
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

	t.Run("FindAssignmentsByAccount", func(t *testing.T) {

		t.Run("Find Existing Account", func(t *testing.T) {
			// Cleanup DB when we're done
			defer truncateAccountAssignmentTable(t, dbSvc)

			// Create a mock assignment
			// with our non-active status
			acctID := "111"
			userID := "222"
			status := db.Active
			timeNow := time.Now().Unix()
			assignment := db.RedboxAccountAssignment{
				AccountID:        acctID,
				UserID:           userID,
				AssignmentStatus: status,
				CreatedOn:        timeNow,
				LastModifiedOn:   timeNow,
			}
			putAssgn, err := dbSvc.PutAccountAssignment(assignment)
			require.Equal(t, db.RedboxAccountAssignment{}, *putAssgn) // should return an empty account assignment since its new
			require.Nil(t, err)

			foundaccount, err := dbSvc.FindAssignmentsByAccount("111")

			require.NotNil(t, foundaccount)
			require.Nil(t, err)
		})

		t.Run("Fail to find non-existent Account", func(t *testing.T) {
			// Cleanup DB when we're done
			defer truncateAccountAssignmentTable(t, dbSvc)

			// Create a mock assignment
			// with our non-active status
			acctID := "333"
			userID := "222"
			status := db.Active
			timeNow := time.Now().Unix()
			assignment := db.RedboxAccountAssignment{
				AccountID:        acctID,
				UserID:           userID,
				AssignmentStatus: status,
				CreatedOn:        timeNow,
				LastModifiedOn:   timeNow,
			}
			putAssgn, err := dbSvc.PutAccountAssignment(assignment)
			require.Equal(t, db.RedboxAccountAssignment{}, *putAssgn) // should return an empty account assignment since its new
			require.Nil(t, err)

			foundassign, err := dbSvc.FindAssignmentsByAccount("111")

			// require.Nil(t, foundassign)
			require.Empty(t, foundassign)
			require.Nil(t, err)
		})
	})

	t.Run("FindAssignmentByUser", func(t *testing.T) {

		t.Run("Find Existing User", func(t *testing.T) {
			// Cleanup DB when we're done
			defer truncateAccountAssignmentTable(t, dbSvc)

			// Create a mock assignment
			// with our non-active status
			acctID := "111"
			userID := "222"
			status := db.Active
			assignment := db.RedboxAccountAssignment{
				AccountID:        acctID,
				UserID:           userID,
				AssignmentStatus: status,
			}
			putAssgn, err := dbSvc.PutAccountAssignment(assignment)
			require.Equal(t, db.RedboxAccountAssignment{}, *putAssgn) // should return an empty account assignment since its new

			foundaccount, err := dbSvc.FindAssignmentByUser("222")

			require.NotNil(t, foundaccount)
			require.Nil(t, err)
		})

		t.Run("Fail to find non-existent Assignment", func(t *testing.T) {
			// Cleanup DB when we're done
			defer truncateAccountAssignmentTable(t, dbSvc)

			// Create a mock assignment
			// with our non-active status
			acctID := "333"
			userID := "222"
			status := db.Active
			assignment := db.RedboxAccountAssignment{
				AccountID:        acctID,
				UserID:           userID,
				AssignmentStatus: status,
			}
			putAssgn, err := dbSvc.PutAccountAssignment(assignment)
			require.Equal(t, db.RedboxAccountAssignment{}, *putAssgn) // should return an empty account assignment since its new
			require.Nil(t, err)

			foundassign, err := dbSvc.FindAssignmentByUser("111")

			require.Nil(t, foundassign)
			require.Nil(t, err)
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
			t.Run("when the account is not assigned", func(t *testing.T) {
				defer truncateAccountTable(t, dbSvc)
				account := *newAccount(accountID, 1561382309)
				err := dbSvc.PutAccount(account)
				require.Nil(t, err, "it returns no errors")
				err = dbSvc.DeleteAccount(accountID)
				require.Nil(t, err, "it returns no errors on delete")
				deletedAccount, err := dbSvc.GetAccount(accountID)
				require.Nil(t, deletedAccount, "the account is deleted")
				require.Nil(t, err, "it returns no errors")
			})

			t.Run("when the account is assigned", func(t *testing.T) {
				defer truncateAccountTable(t, dbSvc)
				account := db.RedboxAccount{
					ID:             accountID,
					AccountStatus:  db.Assigned,
					LastModifiedOn: 1561382309,
				}
				err := dbSvc.PutAccount(account)
				require.Nil(t, err, "it should not error on delete")
				err = dbSvc.DeleteAccount(accountID)
				expectedErrorMessage := fmt.Sprintf("Unable to delete account \"%s\": account is assigned.", accountID)
				require.NotNil(t, err, "it returns an error")
				assert.IsType(t, &db.AccountAssignedError{}, err)
				require.EqualError(t, err, expectedErrorMessage, "it has the correct error message")
			})
		})

		t.Run("when the account does not exists", func(t *testing.T) {
			err := dbSvc.DeleteAccount(accountID)
			require.NotNil(t, err, "it returns an error")
			expectedErrorMessage := fmt.Sprintf("No account found with ID \"%s\".", accountID)
			require.EqualError(t, err, expectedErrorMessage, "it has the correct error message")
			assert.IsType(t, &db.AccountNotFoundError{}, err)
		})
	})
}

func newAccount(id string, timeNow int64) *db.RedboxAccount {
	account := db.RedboxAccount{
		ID:             id,
		AccountStatus:  "Ready",
		LastModifiedOn: timeNow,
	}
	return &account
}

// Remove all records from the RedboxAccount table
func truncateAccountTable(t *testing.T, dbSvc *db.DB) {
	/*
		DynamoDB does not provide a "truncate" method.
		Instead, we need to find all records in the DB table,
		and remove them in a "BatchWrite" requests.
	*/

	// Find all records in the RedboxAccount table
	scanResult, err := dbSvc.Client.Scan(
		&dynamodb.ScanInput{
			TableName: aws.String(dbSvc.AccountTableName),
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
}

/*
Remove all records from the RedboxAccountAssignment table
*/
func truncateAccountAssignmentTable(t *testing.T, dbSvc *db.DB) {
	/*
		DynamoDb does not provide a "truncate" method.
		Instead, we need to find all records in the DB table,
		and remove them in a "BatchWrite" requests.
	*/

	// Find all records in the RedboxAccount table
	scanResult, err := dbSvc.Client.Scan(
		&dynamodb.ScanInput{
			TableName: aws.String(dbSvc.AccountAssignmentTableName),
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
					"AccountId": item["AccountId"],
					"UserId":    item["UserId"],
				},
			},
		})
	}

	// Execute Batch requests, to remove all items
	_, err = dbSvc.Client.BatchWriteItem(
		&dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				dbSvc.AccountAssignmentTableName: deleteRequests,
			},
		},
	)
	require.Nil(t, err)
}
