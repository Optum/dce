// Functional tests because these tests will actually create AWS clients
// and attempt to run basic API calls that list objects or get objects.
package tests

import (
	"testing"

	"github.com/Optum/dce/pkg/config"
	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/aws/aws-sdk-go/service/codebuild/codebuildiface"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider/cognitoidentityprovideriface"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
)

func TestS3Client_FromConfig(t *testing.T) {

	cfgBldr := &config.DCEConfigBuilder{}
	cfgBldr.WithEnv("AWS_CURRENT_REGION", "AWS_CURRENT_REGION", "us-east-1")
	svcBldr := &config.AWSServiceBuilder{Config: cfgBldr}

	_, err := svcBldr.WithS3().Build()
	assert.Nil(t, err)

	// This is what the client code would look like...
	var s3Client s3iface.S3API
	err = svcBldr.Config.GetService(&s3Client)

	// try to use the client to list the bucket
	result, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
	assert.Nil(t, err)
	assert.NotNil(t, result)
}

func TestS3Client_FromConfig_WithBadRegion(t *testing.T) {

	cfgBldr := &config.DCEConfigBuilder{}
	cfgBldr.WithEnv("AWS_CURRENT_REGION", "AWS_CURRENT_REGION", "mars-hemi-1")
	svcBldr := &config.AWSServiceBuilder{Config: cfgBldr}

	_, err := svcBldr.WithS3().Build()
	assert.Nil(t, err)

	// This is what the client code would look like...
	var s3Client s3iface.S3API
	err = svcBldr.Config.GetService(&s3Client)

	// try to use the client to list the bucket
	_, err = s3Client.ListBuckets(&s3.ListBucketsInput{})
	assert.NotNil(t, err)
}

func TestS3ClientAndSNSClient_FromConfig(t *testing.T) {

	cfgBldr := &config.DCEConfigBuilder{}
	cfgBldr.WithEnv("AWS_CURRENT_REGION", "AWS_CURRENT_REGION", "us-east-1")

	svcBldr := &config.AWSServiceBuilder{Config: cfgBldr}

	_, err := svcBldr.
		WithS3().
		WithSNS().
		Build()

	assert.Nil(t, err)

	var s3Client s3iface.S3API
	err = svcBldr.Config.GetService(&s3Client)

	// try to use the client to list the bucket
	result, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
	assert.Nil(t, err)
	assert.NotNil(t, result)

	var snsClient snsiface.SNSAPI
	err = svcBldr.Config.GetService(&snsClient)

	snsResult, err := snsClient.ListTopics(&sns.ListTopicsInput{})
	assert.Nil(t, err)
	assert.NotNil(t, snsResult)

}

func TestAllClients_FromConfig(t *testing.T) {

	cfgBldr := &config.DCEConfigBuilder{}
	cfgBldr.WithEnv("AWS_CURRENT_REGION", "AWS_CURRENT_REGION", "us-east-1")
	svcBldr := &config.AWSServiceBuilder{Config: cfgBldr}

	// Go ahead and just build all of the things...
	_, err := svcBldr.
		WithCodeBuild().
		WithCognito().
		WithDynamoDB().
		WithS3().
		WithSNS().
		WithSQS().
		WithSTS().
		Build()
	assert.Nil(t, err)

	// Now, just for sanity checking, call some sort of list or get on each one...

	t.Run("test code pipeline", func(t *testing.T) {
		var codeBuildClient codebuildiface.CodeBuildAPI
		err = svcBldr.Config.GetService(&codeBuildClient)
		assert.Nil(t, err)

		result, err := codeBuildClient.ListProjects(&codebuild.ListProjectsInput{})
		assert.Nil(t, err)
		assert.NotNil(t, result)
	})

	t.Run("test cognito", func(t *testing.T) {
		var cognitoClient cognitoidentityprovideriface.CognitoIdentityProviderAPI
		err = svcBldr.Config.GetService(&cognitoClient)
		assert.Nil(t, err)

		result, err := cognitoClient.ListUserPools(&cognitoidentityprovider.ListUserPoolsInput{MaxResults: aws.Int64(1)})
		assert.Nil(t, err)
		assert.NotNil(t, result)
	})

	t.Run("test dynamodb", func(t *testing.T) {
		var ddbClient dynamodbiface.DynamoDBAPI
		err = svcBldr.Config.GetService(&ddbClient)
		assert.Nil(t, err)

		result, err := ddbClient.ListTables(&dynamodb.ListTablesInput{})
		assert.Nil(t, err)
		assert.NotNil(t, result)
	})

	t.Run("test s3", func(t *testing.T) {
		var s3Client s3iface.S3API
		err = svcBldr.Config.GetService(&s3Client)
		assert.Nil(t, err)

		result, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
		assert.Nil(t, err)
		assert.NotNil(t, result)
	})

	t.Run("test sns", func(t *testing.T) {
		var snsClient snsiface.SNSAPI
		err = svcBldr.Config.GetService(&snsClient)
		assert.Nil(t, err)

		snsResult, err := snsClient.ListTopics(&sns.ListTopicsInput{})
		assert.Nil(t, err)
		assert.NotNil(t, snsResult)
	})

	t.Run("test sqs", func(t *testing.T) {
		var sqsClient sqsiface.SQSAPI
		err = svcBldr.Config.GetService(&sqsClient)
		assert.Nil(t, err)

		snsResult, err := sqsClient.ListQueues(&sqs.ListQueuesInput{})
		assert.Nil(t, err)
		assert.NotNil(t, snsResult)
	})

	t.Run("test sts", func(t *testing.T) {
		var stsClient stsiface.STSAPI
		err = svcBldr.Config.GetService(&stsClient)
		assert.Nil(t, err)

		snsResult, err := stsClient.GetCallerIdentity(&sts.GetCallerIdentityInput{})
		assert.Nil(t, err)
		assert.NotNil(t, snsResult)
	})

}
