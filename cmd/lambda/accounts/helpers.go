package main

import (
	"bytes"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

func downloadS3Data(s3Client s3iface.S3API, bucket string, key string) ([]byte, error) {
	results, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer results.Body.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, results.Body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
