package testutils

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/aws/aws-sdk-go/service/codebuild/codebuildiface"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/aws/aws-sdk-go/service/sfn/sfniface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func GivenSfnIsEmpty(t *testing.T, svc sfniface.SFNAPI, name string) {
	err := svc.ListExecutionsPages(&sfn.ListExecutionsInput{
		StateMachineArn: aws.String(name),
		StatusFilter:    aws.String("RUNNING"),
	}, func(page *sfn.ListExecutionsOutput, lastPage bool) bool {
		for _, exec := range page.Executions {
			_, err := svc.StopExecution(&sfn.StopExecutionInput{
				ExecutionArn: exec.ExecutionArn,
			})
			require.Nil(t, err)
		}
		return !lastPage
	})
	require.Nil(t, err)
}

func GivenSqsIsEmpty(t *testing.T, svc sqsiface.SQSAPI, sqsURL string) {

	Retry(t, 6, 10*time.Second, func(r *R) {
		atts, err := svc.GetQueueAttributes(&sqs.GetQueueAttributesInput{
			QueueUrl: aws.String(sqsURL),
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

		if total > 0 {
			// if there are more than 0 purge the records
			Retry(t, 2, 30*time.Second, func(r *R) {
				_, err := svc.PurgeQueue(&sqs.PurgeQueueInput{
					QueueUrl: aws.String(sqsURL),
				})
				if err != nil {
					// messages for up to 1 minute aren't guarenteed
					time.Sleep(60 * time.Second)
				}
				assert.Nil(r, err)
			})

			total = 0
		}

		assert.Equalf(r, 0, total, fmt.Sprintf("%d messages in queue %q", total, sqsURL))
	})
}

func GivenCodeBuildIsEmpty(t *testing.T, svc codebuildiface.CodeBuildAPI, codeBuildProject string) {

	ctx := context.Background()
	p := request.Pagination{
		NewRequest: func() (*request.Request, error) {
			req, _ := svc.ListBuildsForProjectRequest(&codebuild.ListBuildsForProjectInput{
				ProjectName: aws.String(codeBuildProject),
			})
			req.SetContext(ctx)
			return req, nil
		},
	}

	for p.Next() {
		page := p.Page().(*codebuild.ListBuildsForProjectOutput)
		builds, _ := svc.BatchGetBuilds(&codebuild.BatchGetBuildsInput{
			Ids: page.Ids,
		})
		for _, build := range builds.Builds {
			if !*build.BuildComplete {
				_, _ = svc.StopBuild(&codebuild.StopBuildInput{
					Id: build.Id,
				})
			}
		}
	}

	require.Nil(t, p.Err())
}
