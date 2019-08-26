package main

import (
	"bytes"
	"io/ioutil"
	"testing"

	awsMocks "github.com/Optum/Redbox/pkg/awsiface/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/require"
)

func TestHelpers_downloadS3Data(t *testing.T) {

	t.Run("Read a file from a S3 Bucket", func(t *testing.T) {

		mockS3 := &awsMocks.S3API{}

		// Should add account to Queue
		mockS3.On("GetObject", &s3.GetObjectInput{
			Bucket: aws.String("test-bucket"),
			Key:    aws.String("test/key.tmpl"),
		},
		).Return(
			&s3.GetObjectOutput{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`Test Data`))),
			}, nil)
		defer mockS3.AssertExpectations(t)

		// Send request
		data, err := downloadS3Data(mockS3, "test-bucket", "test/key.tmpl")
		require.Nil(t, err)
		require.Equal(t, []byte("Test Data"), data)

	})

}
