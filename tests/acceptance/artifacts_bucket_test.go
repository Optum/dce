package tests

import (
	"testing"

	"github.com/stretchr/testify/require"

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
    require.NoError(t, err, "Failed to create AWS session")
    s3Client := s3.New(awsSession)

    // Grab the bucket name from Terraform output
    tfOpts := &terraform.Options{
        TerraformDir: "../../modules",
    }
    bucketName, err := terraform.OutputE(t, tfOpts, "artifacts_bucket_name")
    require.NoError(t, err, "Failed to retrieve artifacts_bucket_name from Terraform output")
    require.NotEmpty(t, bucketName, "artifacts_bucket_name should not be empty")

    t.Run("should be encrypted by default", func(t *testing.T) {
        // Check that the bucket is encrypted
        encryptionOutput, err := s3Client.GetBucketEncryption(
            &s3.GetBucketEncryptionInput{
                Bucket: &bucketName,
            },
        )
        require.NoError(t, err, "Failed to get bucket encryption")
        require.Equal(t,
            "AES256",
            *encryptionOutput.ServerSideEncryptionConfiguration.Rules[0].ApplyServerSideEncryptionByDefault.SSEAlgorithm,
            "Artifacts bucket should be encrypted by default",
        )
    })

    t.Run("should have versioning enabled", func(t *testing.T) {
        // Check that the bucket has versioning enabled
        versioningOutput, err := s3Client.GetBucketVersioning(
            &s3.GetBucketVersioningInput{
                Bucket: &bucketName,
            },
        )
        require.NoError(t, err, "Failed to get bucket versioning")
        require.Equal(t,
            "Enabled",
            *versioningOutput.Status,
            "Artifacts bucket should have versioning enabled",
        )
    })
}
