package api

import (
	"testing"

	"github.com/Optum/dce/tests/testutils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/require"
)

func givenSystemIsEmpty(t *testing.T) {
	testutils.GivenSqsIsEmpty(t, testSvcs.sqsSvc, testConfig.SqsUrl)
	testutils.GivenCodeBuildIsEmpty(t, testSvcs.codeBuildSvc, testConfig.CodeBuildProject)
	testutils.GivenSfnIsEmpty(t, testSvcs.sfnSvc, testConfig.SfnLeaseUsageArn)
	givenAllTablesAreEmpty(t)
}

func givenSystemHasUsageData(t *testing.T) {
	files := []string{
		"../../fixtures/usagelease/user1.json",
	}
	testutils.LoadUsageLeaseRecords(t, testSvcs.dbSvc, testConfig.PrincipalTable, files...)

	files = []string{
		"../../fixtures/usageprincipal/user1.json",
	}
	testutils.LoadUsagePrincipalRecords(t, testSvcs.dbSvc, testConfig.PrincipalTable, files...)
}

func givenAllTablesAreEmpty(t *testing.T) {
	awsSession, err := session.NewSession()
	require.Nil(t, err)
	dbSvc := dynamodb.New(
		awsSession,
		aws.NewConfig().WithRegion(testConfig.AwsRegion),
	)

	testutils.TruncateAccountTable(t, dbSvc, testConfig.AccountTable)
	testutils.TruncateLeaseTable(t, dbSvc, testConfig.LeaseTable)
	testutils.TruncatePrincipalTable(t, dbSvc, testConfig.PrincipalTable)
}
