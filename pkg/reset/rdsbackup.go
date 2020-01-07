package reset

import (
	"log"

	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
)

// RdsService interface for rds backup reset
type RdsService interface {
	DescribeDBInstances(input *rds.DescribeDBInstancesInput) (*rds.DescribeDBInstancesOutput, error)
	ModifyDBInstance(input *rds.ModifyDBInstanceInput) (*rds.ModifyDBInstanceOutput, error)
	DescribeDBInstanceAutomatedBackups(input *rds.DescribeDBInstanceAutomatedBackupsInput) (*rds.DescribeDBInstanceAutomatedBackupsOutput, error)
	DeleteDBInstanceAutomatedBackup(input *rds.DeleteDBInstanceAutomatedBackupInput) (*rds.DeleteDBInstanceAutomatedBackupOutput, error)
}

// RdsReset struct implements above interface
type RdsReset struct {
	Client rdsiface.RDSAPI
}

// DescribeDBInstances implementation
func (r RdsReset) DescribeDBInstances(input *rds.DescribeDBInstancesInput) (*rds.DescribeDBInstancesOutput, error) {
	return r.Client.DescribeDBInstances(input)
}

// ModifyDBInstance implementation
func (r RdsReset) ModifyDBInstance(input *rds.ModifyDBInstanceInput) (*rds.ModifyDBInstanceOutput, error) {
	return r.Client.ModifyDBInstance(input)
}

// DescribeDBInstanceAutomatedBackups implementation
func (r RdsReset) DescribeDBInstanceAutomatedBackups(input *rds.DescribeDBInstanceAutomatedBackupsInput) (*rds.DescribeDBInstanceAutomatedBackupsOutput, error) {
	return r.Client.DescribeDBInstanceAutomatedBackups(input)
}

// DeleteDBInstanceAutomatedBackup implementation
func (r RdsReset) DeleteDBInstanceAutomatedBackup(input *rds.DeleteDBInstanceAutomatedBackupInput) (*rds.DeleteDBInstanceAutomatedBackupOutput, error) {
	return r.Client.DeleteDBInstanceAutomatedBackup(input)
}

// DeleteRdsBackups deletes RDS backups
func DeleteRdsBackups(rdsService RdsService) error {

	var retentionPeriod int64
	var applyNow = true

	describeDBInstancesInput := &rds.DescribeDBInstancesInput{}

	dbInstances, err := rdsService.DescribeDBInstances(describeDBInstancesInput)
	if err != nil {
		return err
	}

	for _, dbInstance := range dbInstances.DBInstances {
		log.Printf("Modify dbInstance retention period to 0 for %s \n", *dbInstance.DBInstanceArn)

		modifyDBInstanceInput := &rds.ModifyDBInstanceInput{
			DBInstanceIdentifier:  dbInstance.DBInstanceIdentifier,
			BackupRetentionPeriod: &retentionPeriod,
			ApplyImmediately:      &applyNow,
		}
		_, err := rdsService.ModifyDBInstance(modifyDBInstanceInput)
		if err != nil {
			return err
		}

		describeDBInstanceAutomatedBackupInput := &rds.DescribeDBInstanceAutomatedBackupsInput{
			DbiResourceId: dbInstance.DbiResourceId,
		}

		dbInstanceBackups, err := rdsService.DescribeDBInstanceAutomatedBackups(describeDBInstanceAutomatedBackupInput)
		if err != nil {
			return err
		}
		for _, dbInstanceBackup := range dbInstanceBackups.DBInstanceAutomatedBackups {
			log.Printf("dbInstanceBackup to be deleted : %s \n", *dbInstanceBackup.DbiResourceId)
			deleteDBInstanceAutomatedBackupInput := &rds.DeleteDBInstanceAutomatedBackupInput{
				DbiResourceId: dbInstanceBackup.DbiResourceId,
			}

			// Ignoring errors, since changing retention period to 0 takes care of deleting automated backups.
			_, _ = rdsService.DeleteDBInstanceAutomatedBackup(deleteDBInstanceAutomatedBackupInput)
		}

	}
	return nil
}
