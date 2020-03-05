package api

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/Optum/dce/tests/testutils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func truncateTables(t *testing.T) {
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

func whenSfnIsEmpty(t *testing.T) {
	err := testSvcs.sfnSvc.ListExecutionsPages(&sfn.ListExecutionsInput{
		StateMachineArn: aws.String(testConfig.SfnLeaseUsageArn),
		StatusFilter:    aws.String("RUNNING"),
	}, func(page *sfn.ListExecutionsOutput, lastPage bool) bool {
		for _, exec := range page.Executions {
			_, err := testSvcs.sfnSvc.StopExecution(&sfn.StopExecutionInput{
				ExecutionArn: exec.ExecutionArn,
			})
			require.Nil(t, err)
		}
		return !lastPage
	})
	require.Nil(t, err)
}

func whenSqsIsEmpty(t *testing.T) {

	testutils.Retry(t, 60, 2*time.Second, func(r *testutils.R) {
		atts, err := testSvcs.sqsSvc.GetQueueAttributes(&sqs.GetQueueAttributesInput{
			QueueUrl: aws.String(testConfig.SqsUrl),
			AttributeNames: []*string{
				aws.String("ApproximateNumberOfMessages"),
				aws.String("ApproximateNumberOfMessagesDelayed"),
				aws.String("ApproximateNumberOfMessagesNotVisible"),
			},
		})
		assert.Nil(r, err)

		var total int = 0
		for attrib := range atts.Attributes {
			prop := atts.Attributes[attrib]
			i, _ := strconv.Atoi(*prop)
			total = total + i
		}

		assert.Equalf(r, 0, total, fmt.Sprintf("%d messages in queue %q", total, testConfig.SqsUrl))
	})
}

func whenCodeBuildIsEmpty(t *testing.T) {

	ctx := context.Background()
	p := request.Pagination{
		NewRequest: func() (*request.Request, error) {
			req, _ := testSvcs.codeBuildSvc.ListBuildsForProjectRequest(&codebuild.ListBuildsForProjectInput{
				ProjectName: aws.String(testConfig.CodeBuildProject),
			})
			req.SetContext(ctx)
			return req, nil
		},
	}

	for p.Next() {
		page := p.Page().(*codebuild.ListBuildsForProjectOutput)
		builds, _ := testSvcs.codeBuildSvc.BatchGetBuilds(&codebuild.BatchGetBuildsInput{
			Ids: page.Ids,
		})
		for _, build := range builds.Builds {
			if !*build.BuildComplete {
				testSvcs.codeBuildSvc.StopBuild(&codebuild.StopBuildInput{
					Id: build.Id,
				})
			}
		}
	}

	require.Nil(t, p.Err())
}

func whenSystemIsEmpty(t *testing.T) {
	whenSqsIsEmpty(t)
	whenCodeBuildIsEmpty(t)
	whenSfnIsEmpty(t)
	truncateTables(t)
}

func whenSystemHasUsageData(t *testing.T) {
	truncateTables(t)

}
