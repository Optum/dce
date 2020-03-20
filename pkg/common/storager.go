package common

import (
	"bytes"
	"html/template"
	"log"
	"os"
	"strings"

	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// Storager interface requires a method to delete an S3 Object from a Bucket
// based on the provided S3 Object Input
type Storager interface {
	GetObject(bucket string, key string) (string, error)
	GetTemplateObject(bucket string, key string, input interface{}) (string, string, error)
	Download(bukcet string, key string, filepath string) error
}

// S3 implements the Storage interface using AWS S3 Client
type S3 struct {
	Client  *s3.S3
	Manager *s3manager.Downloader
}

// GetObject returns a string output based on the results of the retrieval
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

// GetObjectWithETag returns a string output based on the results of the retrieval
// of an existing object from S3
func (stor S3) GetObjectWithETag(bucket string, key string) (string, string, error) {
	// Retrieve the S3 Object
	getInput := s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}
	getOutput, err := stor.Client.GetObject(&getInput)
	if err != nil {
		return "", "", err
	}

	// Convert to string
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(getOutput.Body)
	if err != nil {
		return "", "", err
	}
	object := buf.String()

	return object, *getOutput.ETag, nil
}

// GetTemplateObject returns a string output based on the results of the retrieval
// of an existing object from S3
func (stor S3) GetTemplateObject(bucket string, key string, input interface{}) (string, string, error) {
	// Retrieve the S3 Object
	templateString, templateETag, err := stor.GetObjectWithETag(bucket, key)
	if err != nil {
		return "", "", errors.NewInternalServer("unexpected failure getting template", err)
	}

	tmpl := template.New(key)

	// Add `StringsJoin` function to template
	// See https://stackoverflow.com/a/42724991
	tmpl = tmpl.Funcs(template.FuncMap{"StringsJoin": strings.Join})

	templParsed, err := tmpl.Parse(templateString)
	if err != nil {
		return "", "", errors.NewInternalServer("unexpected failure parsing template", err)
	}

	// Render template
	buf := &bytes.Buffer{}
	err1 := templParsed.Execute(buf, input)

	return strings.TrimSpace(buf.String()), templateETag, err1
}

// Download downloads an S3 Bucket object to the file path provided and returns
// any errors if any
func (stor S3) Download(bucket string, key string, filepath string) error {
	// Initialize the file, verify the path is valid
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Set up the download inputs and execute
	getInput := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}
	_, err = stor.Manager.Download(file, getInput)
	return err
}
