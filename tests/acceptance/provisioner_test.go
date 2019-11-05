package tests

import (
	"testing"
	"time"

	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/provision"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
)

func TestProvisioner(t *testing.T) {
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
		tfOut["redbox_lease_db_table_name"].(string),
		7,
	)

	dbSvc.ConsistendRead = true

	// Configure the Provisioner service
	provSvc := provision.AccountProvision{
		DBSvc: dbSvc,
	}

	t.Run("Activate Account Lease", func(t *testing.T) {
		t.Run("Should Create a New Account Lease", func(t *testing.T) {
			// Cleanup table on completion
			defer truncateLeaseTable(t, dbSvc)

			sevenDaysFromNow := time.Now().AddDate(0, 0, 7).Unix()

			// Activate the below Account Lease
			acctID := "111"
			principalID := "222"
			var budgetAmount float64 = 300
			budgetCurrency := "USD"
			budgetNotificationEmails := []string{"test@test.com"}
			result, err := provSvc.ActivateAccount(true, principalID,
				acctID, budgetAmount, budgetCurrency, budgetNotificationEmails, sevenDaysFromNow)

			// Verify the lease returned
			require.Nil(t, err)
			require.Equal(t, principalID, result.PrincipalID)
			require.Equal(t, acctID, result.AccountID)
			require.Equal(t, db.Active, result.LeaseStatus)
			require.Equal(t, db.LeaseActive, result.LeaseStatusReason)
			require.NotEqual(t, 0, result.CreatedOn)
			require.NotEqual(t, 0, result.LastModifiedOn)
			require.NotEqual(t, 0, result.LastModifiedOn)

			// Get the lease
			assgnAfter, err := dbSvc.GetLease(acctID, principalID)

			// Verify the lease exists
			require.Nil(t, err)
			require.Equal(t, result.PrincipalID, assgnAfter.PrincipalID)
			require.Equal(t, result.AccountID, assgnAfter.AccountID)
			require.Equal(t, result.LeaseStatus,
				assgnAfter.LeaseStatus)
			require.Equal(t, result.LeaseStatusReason,
				assgnAfter.LeaseStatusReason)
			require.Equal(t, result.CreatedOn, assgnAfter.CreatedOn)
			require.Equal(t, result.LastModifiedOn, assgnAfter.LastModifiedOn)
			require.Equal(t, result.LeaseStatusModifiedOn, assgnAfter.LeaseStatusModifiedOn)
		})

		t.Run("Should Transition Existing Lease", func(t *testing.T) {
			// Cleanup table on completion
			defer truncateLeaseTable(t, dbSvc)

			// Put an Lease into the table to be transitioned.
			acctID := "111"
			principalID := "222"
			var budgetAmount float64 = 300
			budgetCurrency := "USD"
			budgetNotificationEmails := []string{"test@test.com"}
			sevenDaysFromNow := time.Now().AddDate(0, 0, 7).Unix()

			timeNow := time.Now().Unix()
			assgn := db.RedboxLease{
				ID:                    uuid.New().String(),
				AccountID:             acctID,
				PrincipalID:           principalID,
				LeaseStatus:           db.Active,
				CreatedOn:             timeNow,
				LastModifiedOn:        timeNow,
				LeaseStatusModifiedOn: timeNow,
			}
			putAssgn, err := dbSvc.PutLease(assgn)
			require.Nil(t, err)
			require.Equal(t, db.RedboxLease{}, *putAssgn) // should return an empty account lease since its new

			// Activate the below Account Lease
			result, err := provSvc.ActivateAccount(true, principalID,
				acctID, budgetAmount, budgetCurrency, budgetNotificationEmails, sevenDaysFromNow)

			// Verify the lease returned
			require.Equal(t, principalID, result.PrincipalID)
			require.Equal(t, acctID, result.AccountID)
			require.Equal(t, db.Active, result.LeaseStatus)
			require.NotEqual(t, 0, result.CreatedOn)
			require.NotEqual(t, 0, result.LastModifiedOn)
			require.NotEqual(t, 0, result.LeaseStatusModifiedOn)
			require.Nil(t, err)

			// Get the lease
			assgnAfter, err := dbSvc.GetLease(acctID, principalID)

			// Verify the lease exists
			require.Equal(t, result.PrincipalID, assgnAfter.PrincipalID)
			require.Equal(t, result.AccountID, assgnAfter.AccountID)
			require.Equal(t, result.LeaseStatus,
				assgnAfter.LeaseStatus)
			require.Equal(t, result.CreatedOn, assgnAfter.CreatedOn)
			require.Equal(t, result.LastModifiedOn, assgnAfter.LastModifiedOn)
			require.Equal(t, result.LeaseStatusModifiedOn, assgnAfter.LeaseStatusModifiedOn)
			require.Nil(t, err)
		})

	})
}
