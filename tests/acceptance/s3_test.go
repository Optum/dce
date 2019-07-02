package tests

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/Optum/Redbox/pkg/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
)

func TestS3(t *testing.T) {
	// Load Terraform outputs
	tfOpts := &terraform.Options{
		TerraformDir: "../../modules",
	}
	tfOut := terraform.OutputAll(t, tfOpts)

	// Initialize variables
	artifactsFilePath := "testdata/s3-test-file.txt"
	artifactsBucket := tfOut["artifacts_bucket_name"].(string)
	artifactsDir := "s3-test-dir"
	artifactsKey := fmt.Sprintf("%s/test.txt", artifactsDir)
	artifactsBody, err := ioutil.ReadFile(artifactsFilePath)
	require.Nil(t, err)

	// Configure the S3 Service
	awsSession, err := session.NewSession(
		aws.NewConfig().WithRegion(tfOut["aws_region"].(string)))
	require.Nil(t, err)
	s3Svc := &common.S3{
		Client:  s3.New(awsSession),
		Manager: s3manager.NewDownloader(awsSession),
	}

	// Test GetObject
	t.Run("GetObject", func(t *testing.T) {
		t.Run("Should retrieve an existing object in a bucket", func(t *testing.T) {
			// Put the test file into the artifact bucket for testing
			_, err := s3Svc.Client.PutObject(
				&s3.PutObjectInput{
					Bucket:               &artifactsBucket,
					Key:                  &artifactsKey,
					Body:                 aws.ReadSeekCloser(bytes.NewReader(artifactsBody)),
					ServerSideEncryption: aws.String("AES256"),
				},
			)
			require.Nil(t, err)
			defer removeObject(t, s3Svc, &artifactsBucket, &artifactsKey)

			// Get the test file from the artifact bucket and verify
			body, err := s3Svc.GetObject(artifactsBucket, artifactsKey)
			require.Nil(t, err)
			require.Equal(t, string(artifactsBody), body)
		})
	})

	// Test Upload
	t.Run("Upload", func(t *testing.T) {
		t.Run("Should put an object to the bucket", func(t *testing.T) {
			// Put the test file into the artifact bucket
			err := s3Svc.Upload(artifactsBucket, artifactsKey,
				artifactsFilePath)
			require.Nil(t, err)
			defer removeObject(t, s3Svc, &artifactsBucket, &artifactsKey)

			// Get the test file from the artifact bucket and verify
			obj, err := s3Svc.Client.GetObject(
				&s3.GetObjectInput{
					Bucket: &artifactsBucket,
					Key:    &artifactsKey,
				},
			)
			require.Nil(t, err)
			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(obj.Body)
			require.Nil(t, err)
			require.Equal(t, string(artifactsBody), buf.String())
		})
	})

	// Test Download
	t.Run("Download", func(t *testing.T) {
		t.Run("Should download an object from a bucket as a file", func(t *testing.T) {
			// Put the test file into the artifact bucket for testing
			_, err := s3Svc.Client.PutObject(
				&s3.PutObjectInput{
					Bucket:               &artifactsBucket,
					Key:                  &artifactsKey,
					Body:                 aws.ReadSeekCloser(bytes.NewReader(artifactsBody)),
					ServerSideEncryption: aws.String("AES256"),
				},
			)
			require.Nil(t, err)
			defer removeObject(t, s3Svc, &artifactsBucket, &artifactsKey)

			// Get the test file from the artifact bucket and verify
			localFile := "/tmp/local-s3-file.txt"
			err = s3Svc.Download(artifactsBucket, artifactsKey, localFile)
			require.Nil(t, err)
			defer os.Remove(localFile)
			localBody, err := ioutil.ReadFile("/tmp/local-s3-file.txt")
			require.Nil(t, err)
			require.Equal(t, artifactsBody, localBody)
		})
	})
}

// removeObject is a helper function to clean up after each S3 test by deleting
// the provided bucket object
func removeObject(t *testing.T, s3Svc *common.S3, bucket *string, key *string) {
	_, err := s3Svc.Client.DeleteObject(
		&s3.DeleteObjectInput{
			Bucket: bucket,
			Key:    key,
		},
	)
	require.Nil(t, err)
}
