package tests

import (
	"testing"
	"time"

	"github.com/Optum/Redbox/pkg/db"
	"github.com/Optum/Redbox/pkg/provision"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
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
		tfOut["dynamodb_table_account_assignment_name"].(string),
	)

	// Configure the Provisioner service
	provSvc := provision.AccountProvision{
		DBSvc: dbSvc,
	}

	t.Run("Activate Account Assignment", func(t *testing.T) {
		t.Run("Should Create a New Account Assignment", func(t *testing.T) {
			// Cleanup table on completion
			defer truncateAccountAssignmentTable(t, dbSvc)

			// Activate the below Account Assignment
			acctID := "111"
			userID := "222"
			result, err := provSvc.ActivateAccountAssignment(true, userID,
				acctID)

			// Verify the assignment returned
			require.Equal(t, userID, result.UserID)
			require.Equal(t, acctID, result.AccountID)
			require.Equal(t, db.Active, result.AssignmentStatus)
			require.NotEqual(t, 0, result.CreatedOn)
			require.NotEqual(t, 0, result.LastModifiedOn)
			require.Nil(t, err)

			// Get the assignment
			assgnAfter, err := dbSvc.GetAssignment(acctID, userID)

			// Verify the assignment exists
			require.Equal(t, result.UserID, assgnAfter.UserID)
			require.Equal(t, result.AccountID, assgnAfter.AccountID)
			require.Equal(t, result.AssignmentStatus,
				assgnAfter.AssignmentStatus)
			require.Equal(t, result.CreatedOn, assgnAfter.CreatedOn)
			require.Equal(t, result.LastModifiedOn, assgnAfter.LastModifiedOn)
			require.Nil(t, err)
		})

		t.Run("Should Transition Existing Assignment", func(t *testing.T) {
			// Cleanup table on completion
			defer truncateAccountAssignmentTable(t, dbSvc)

			// Put an Assignment into the table to be transitioned.
			acctID := "111"
			userID := "222"
			timeNow := time.Now().Unix()
			assgn := db.RedboxAccountAssignment{
				AccountID:        acctID,
				UserID:           userID,
				AssignmentStatus: db.Active,
				CreatedOn:        timeNow,
				LastModifiedOn:   timeNow,
			}
			putAssgn, err := dbSvc.PutAccountAssignment(assgn)
			require.Equal(t, db.RedboxAccountAssignment{}, *putAssgn) // should return an empty account assignment since its new
			require.Nil(t, err)

			// Activate the below Account Assignment
			result, err := provSvc.ActivateAccountAssignment(true, userID,
				acctID)

			// Verify the assignment returned
			require.Equal(t, userID, result.UserID)
			require.Equal(t, acctID, result.AccountID)
			require.Equal(t, db.Active, result.AssignmentStatus)
			require.NotEqual(t, 0, result.CreatedOn)
			require.NotEqual(t, 0, result.LastModifiedOn)
			require.Nil(t, err)

			// Get the assignment
			assgnAfter, err := dbSvc.GetAssignment(acctID, userID)

			// Verify the assignment exists
			require.Equal(t, result.UserID, assgnAfter.UserID)
			require.Equal(t, result.AccountID, assgnAfter.AccountID)
			require.Equal(t, result.AssignmentStatus,
				assgnAfter.AssignmentStatus)
			require.Equal(t, result.CreatedOn, assgnAfter.CreatedOn)
			require.Equal(t, result.LastModifiedOn, assgnAfter.LastModifiedOn)
			require.Nil(t, err)
		})

	})
}
