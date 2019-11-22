// Functional tests because these tests will actually create AWS clients
// and attempt to run basic API calls that list objects or get objects.
package tests

import (
	"testing"

	"github.com/Optum/dce/pkg/config"
	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
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
