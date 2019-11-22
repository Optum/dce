package config

import (
	"log"
	"reflect"
	"runtime"

	"github.com/aws/aws-sdk-go/aws/session"
)

// AWSSessionKey is the key for the configuration for the AWS session
const AWSSessionKey = "AWSSession"

// AWSConfigurationError is returned when an AWS service cannot be properly configured.
type AWSConfigurationError error

// AWSServiceBuilder creates the AWSServices structure, which contains references
// to the AWS services.
type AWSServiceBuilder interface {
	// TODO: Session might need to stay internal. Not sure why it would
	// NOT be ever called.
	// WithSession tells the builder to add an AWS session to the result
	// WithSession() *AWSServiceBuilder
	// WithSTS tells the builder to add an AWS STS service to the `Configurater`
	WithSTS() *AWSServiceBuilder
	// WithSNS tells the builder to add an AWS SNS service to the `Configurater`
	WithSNS() *AWSServiceBuilder
	// WithSQS tells the builder to add an AWS SQS service to the `Configurater`
	WithSQS() *AWSServiceBuilder
	// WithDynamoDB tells the builder to add an AWS DynamoDB service to the `Configurater`
	WithDynamoDB() *AWSServiceBuilder
	// WithS3 tells the builder to add an AWS S3 service to the `Configurater`
	WithS3() *AWSServiceBuilder
	// WithCognito tells the builder to add an AWS Cognito service to the `Configurater`
	WithCognito() *AWSServiceBuilder
	// WithCodePipeline tells the builder to add an AWS CodePipeline service to the `Configurater`
	WithCodePipeline() *AWSServiceBuilder
	// Build creates and returns a structue with AWS services
	Build() (*Configurater, error)
}

// createrFunc internal functions for handling the creation of the services
type createrFunc func(config *Configurater) error

// DefaultAWSServiceBuilder is the default implementation of the `AWSServiceBuilder`
type DefaultAWSServiceBuilder struct {
	handlers []createrFunc
	Config   *Configurater
}

// WithSession tells the builder to add an AWS session to the result
// func (bldr *DefaultAWSServiceBuilder) WithSession() *DefaultAWSServiceBuilder {
// 	bldr.handlers = append(bldr.handlers, createSession)
// 	return bldr
// }

// WithSTS tells the builder to add an AWS STS service to the `Configurater`
func (bldr *DefaultAWSServiceBuilder) WithSTS() *DefaultAWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, createSTS)
	return bldr
}

// WithSNS tells the builder to add an AWS SNS service to the `Configurater`
func (bldr *DefaultAWSServiceBuilder) WithSNS() *DefaultAWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, createSNS)
	return bldr
}

// WithSQS tells the builder to add an AWS SQS service to the `Configurater`
func (bldr *DefaultAWSServiceBuilder) WithSQS() *DefaultAWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, createSQS)
	return bldr
}

// WithDynamoDB tells the builder to add an AWS DynamoDB service to the `Configurater`
func (bldr *DefaultAWSServiceBuilder) WithDynamoDB() *DefaultAWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, createDynamoDB)
	return bldr
}

// WithS3 tells the builder to add an AWS S3 service to the `Configurater`
func (bldr *DefaultAWSServiceBuilder) WithS3() *DefaultAWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, createS3)
	return bldr
}

// WithCognito tells the builder to add an AWS Cognito service to the `Configurater`
func (bldr *DefaultAWSServiceBuilder) WithCognito() *DefaultAWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, createCognito)
	return bldr
}

// WithCodePipeline tells the builder to add an AWS CodePipeline service to the `Configurater`
func (bldr *DefaultAWSServiceBuilder) WithCodePipeline() *DefaultAWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, createCodePipeline)
	return bldr
}

// Build creates and returns a structue with AWS services
func (bldr *DefaultAWSServiceBuilder) Build() (*Configurater, error) {

	// Create session is done first, and explicitly, because everything else
	// uses it
	err := createSession(bldr.Config)

	if err != nil {
		log.Printf("Could not create session: %s", err.Error())
		return bldr.Config, AWSConfigurationError(err)
	}

	for _, f := range bldr.handlers {
		err := f(bldr.Config)
		if err != nil {
			log.Printf("Error while trying to execute handler: %s", runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name())
			// TODO: Do we want to keep going or stop at the first error?
			return bldr.Config, AWSConfigurationError(err)
		}
	}

	return bldr.Config, nil
}

func createSession(config *Configurater) error {
	awsSession, err := session.NewSession()
	config.WithVal(AWSSessionKey, awsSession)
	return err
}

func createSTS(config *Configurater) error {
	// TODO: Add code in here to create the AWS session and add it to the configuration
	return nil
}

func createSNS(config *Configurater) error {
	// TODO: Add code in here to create the AWS session and add it to the configuration
	return nil
}

func createSQS(config *Configurater) error {
	// TODO: Add code in here to create the AWS session and add it to the configuration
	return nil
}

func createDynamoDB(config *Configurater) error {
	// TODO: Add code in here to create the AWS session and add it to the configuration
	return nil
}

func createS3(config *Configurater) error {
	// TODO: Add code in here to create the AWS session and add it to the configuration
	return nil
}

func createCognito(config *Configurater) error {
	// TODO: Add code in here to create the AWS session and add it to the configuration
	return nil
}

func createCodePipeline(config *Configurater) error {
	// TODO: Add code in here to create the AWS session and add it to the configuration
	return nil
}
