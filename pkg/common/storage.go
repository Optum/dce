package common

import (
	"bytes"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Storager interface requires a method to delete an S3 Object from a Bucket
// based on the provided S3 Object Input
type Storager interface {
	GetObject(string, string) (string, error)
	PutObject(string, string, io.ReadSeeker) error
}

// S3 implements the Storage interface using AWS S3 Client
type S3 struct {
	Client *s3.S3
}

// GetObject returns S3 GetObject Output based on the results of the retrieval
// of an existing object from S3
func (stor S3) GetObject(bucket string, key string) (string, error) {
	// Retrieve the S3 Object
	getInput := s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}
	getOutput, err := stor.Client.GetObject(&getInput)
	if err != nil {
		return "", err
	}

	// Convert to string
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(getOutput.Body)
	if err != nil {
		return "", err
	}
	object := buf.String()

	return object, nil
}

// PutObject returns S3 PutObject Output based on the results of the put action
// of an object to S3
func (stor S3) PutObject(bucket string, key string, body io.ReadSeeker) error {
	putInput := s3.PutObjectInput{
		Bucket:               &bucket,
		Key:                  &key,
		Body:                 body,
		ServerSideEncryption: aws.String("AES256"),
	}
	_, err := stor.Client.PutObject(&putInput)
	return err
}
