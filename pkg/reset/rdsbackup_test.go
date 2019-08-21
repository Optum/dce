package reset

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/rds/rdsiface"
	"github.com/stretchr/testify/assert"
)

type mockRdsReset struct {
	rdsiface.RDSAPI
}

// DescribeDBInstances mock implementation
func (r mockRdsReset) DescribeDBInstances(input *rds.DescribeDBInstancesInput) (*rds.DescribeDBInstancesOutput, error) {
	db1 := &rds.DBInstance{
		DbiResourceId:        aws.String("test1"),
		DBInstanceIdentifier: aws.String("test1_id"),
		DBInstanceArn:        aws.String("test1_arn"),
	}
	mockOutput := &rds.DescribeDBInstancesOutput{
		DBInstances: []*rds.DBInstance{db1},
	}
	return mockOutput, nil
}

// ModifyDBInstance mock implementation
func (r mockRdsReset) ModifyDBInstance(input *rds.ModifyDBInstanceInput) (*rds.ModifyDBInstanceOutput, error) {
	mockOutput := &rds.ModifyDBInstanceOutput{}
	return mockOutput, nil
}

// DescribeDBInstanceAutomatedBackups mock implementation
func (r mockRdsReset) DescribeDBInstanceAutomatedBackups(input *rds.DescribeDBInstanceAutomatedBackupsInput) (*rds.DescribeDBInstanceAutomatedBackupsOutput, error) {

	db1Backup1 := &rds.DBInstanceAutomatedBackup{
		DbiResourceId: aws.String("test1"),
	}
	mockOutput := &rds.DescribeDBInstanceAutomatedBackupsOutput{
		DBInstanceAutomatedBackups: []*rds.DBInstanceAutomatedBackup{db1Backup1},
	}
	return mockOutput, nil
}

// DeleteDBInstanceAutomatedBackup mock implementation
func (r mockRdsReset) DeleteDBInstanceAutomatedBackup(input *rds.DeleteDBInstanceAutomatedBackupInput) (*rds.DeleteDBInstanceAutomatedBackupOutput, error) {
	mockOutput := &rds.DeleteDBInstanceAutomatedBackupOutput{}
	return mockOutput, nil
}

func TestDeleteRdsBackups(t *testing.T) {
	mockRds := new(mockRdsReset)
	err := DeleteRdsBackups(mockRds)
	assert.Nil(t, err, "There should be no errors")
}
