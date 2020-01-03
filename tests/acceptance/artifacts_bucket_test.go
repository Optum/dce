package tests

import (
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gruntwork-io/terratest/modules/terraform"
)

func TestArtifactsBucket(t *testing.T) {
	// Create an S3 client
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(endpoints.UsEast1RegionID),
	})
	require.Nil(t, err)
	s3Client := s3.New(awsSession)

	// Grab the bucket name from Terraform output
	tfOpts := &terraform.Options{
		TerraformDir: "../../modules",
	}
	bucketName := terraform.Output(t, tfOpts, "artifacts_bucket_name")

	t.Run("should be encrypted by default", func(t *testing.T) {
		// Check that the bucket is encrypted
		encryptionOutput, err := s3Client.GetBucketEncryption(
			&s3.GetBucketEncryptionInput{
				Bucket: &bucketName,
			},
		)
		require.Nil(t, err)
		require.Equal(t,
			*encryptionOutput.ServerSideEncryptionConfiguration.Rules[0].ApplyServerSideEncryptionByDefault.SSEAlgorithm,
			"AES256",
			"Artifacts bucket should be encrypted by default",
		)
	})

	t.Run("should log to itself", func(t *testing.T) {
		loggingOutput, err := s3Client.GetBucketLogging(
			&s3.GetBucketLoggingInput{
				Bucket: &bucketName,
			},
		)
		require.Nil(t, err)
		require.Equal(t,
			*loggingOutput.LoggingEnabled.TargetBucket,
			bucketName,
			"Target log bucket should be itself",
		)
		require.Equal(t,
			*loggingOutput.LoggingEnabled.TargetPrefix,
			"logs/",
			"Should log to the /logs prefix",
		)
	})

	t.Run("should have versioning enabled", func(t *testing.T) {
		versioningOutput, err := s3Client.GetBucketVersioning(
			&s3.GetBucketVersioningInput{
				Bucket: &bucketName,
			},
		)
		require.Nil(t, err)
		require.Equal(t, *versioningOutput.Status, "Enabled")
	})

}
