package config

import (
	"log"
	"reflect"
	"runtime"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
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

// AWSSessionKey is the key for the configuration for the AWS session
const AWSSessionKey = "AWSSession"

// AWSConfigurationError is returned when an AWS service cannot be properly configured.
type AWSConfigurationError error

// createrFunc internal functions for handling the creation of the services
type createrFunc func(config *ConfigurationBuilder) error

// AWSServiceBuilder is the default implementation of the `AWSServiceBuilder`
type AWSServiceBuilder struct {
	handlers   []createrFunc
	awsSession *session.Session
	Config     *ConfigurationBuilder
}

// WithSTS tells the builder to add an AWS STS service to the `DefaultConfigurater`
func (bldr *AWSServiceBuilder) WithSTS() *AWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createSTS)
	return bldr
}

// WithSNS tells the builder to add an AWS SNS service to the `DefaultConfigurater`
func (bldr *AWSServiceBuilder) WithSNS() *AWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createSNS)
	return bldr
}

// WithSQS tells the builder to add an AWS SQS service to the `DefaultConfigurater`
func (bldr *AWSServiceBuilder) WithSQS() *AWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createSQS)
	return bldr
}

// WithDynamoDB tells the builder to add an AWS DynamoDB service to the `DefaultConfigurater`
func (bldr *AWSServiceBuilder) WithDynamoDB() *AWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createDynamoDB)
	return bldr
}

// WithS3 tells the builder to add an AWS S3 service to the `DefaultConfigurater`
func (bldr *AWSServiceBuilder) WithS3() *AWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createS3)
	return bldr
}

// WithCognito tells the builder to add an AWS Cognito service to the `DefaultConfigurater`
func (bldr *AWSServiceBuilder) WithCognito() *AWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createCognito)
	return bldr
}

// WithCodeBuild tells the builder to add an AWS CodeBuild service to the `DefaultConfigurater`
func (bldr *AWSServiceBuilder) WithCodeBuild() *AWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createCodeBuild)
	return bldr
}

// Build creates and returns a structue with AWS services
func (bldr *AWSServiceBuilder) Build() (*ConfigurationBuilder, error) {

	err := bldr.Config.Build()
	if err != nil {
		// We failed to build the configuration, so honestly there is no
		// point in continuating...
		return bldr.Config, AWSConfigurationError(err)
	}

	// Create session is done first, and explicitly, because everything else
	// uses it
	err = bldr.createSession(bldr.Config)

	if err != nil {
		log.Printf("Could not create session: %s", err.Error())
		return bldr.Config, AWSConfigurationError(err)
	}

	for _, f := range bldr.handlers {
		err := f(bldr.Config)
		if err != nil {
			log.Printf("Error while trying to execute handler: %s", runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name())
			return bldr.Config, AWSConfigurationError(err)
		}
	}

	// make certain build is called before returning.
	return bldr.Config, nil
}

func (bldr *AWSServiceBuilder) createSession(config *ConfigurationBuilder) error {
	var err error
	region, err := bldr.Config.GetStringVal("AWS_CURRENT_REGION")
	if err == nil {
		log.Printf("Using AWS region \"%s\" to create session...", region)
		bldr.awsSession, err = session.NewSession(
			&aws.Config{
				Region: aws.String(region),
			},
		)
	} else {
		log.Println("Creating AWS session using defaults...")
		bldr.awsSession, err = session.NewSession()
	}
	return err
}

func (bldr *AWSServiceBuilder) createSTS(config *ConfigurationBuilder) error {
	var stsSvc stsiface.STSAPI
	stsSvc = sts.New(bldr.awsSession)
	config.WithService(stsSvc)
	return nil
}

func (bldr *AWSServiceBuilder) createSNS(config *ConfigurationBuilder) error {
	var snsSvc snsiface.SNSAPI
	snsSvc = sns.New(bldr.awsSession)
	config.WithService(snsSvc)
	return nil
}

func (bldr *AWSServiceBuilder) createSQS(config *ConfigurationBuilder) error {
	var sqsSvc sqsiface.SQSAPI
	sqsSvc = sqs.New(bldr.awsSession)
	config.WithService(sqsSvc)
	return nil
}

func (bldr *AWSServiceBuilder) createDynamoDB(config *ConfigurationBuilder) error {
	var dynamodbSvc dynamodbiface.DynamoDBAPI
	dynamodbSvc = dynamodb.New(bldr.awsSession)
	config.WithService(dynamodbSvc)
	return nil
}

func (bldr *AWSServiceBuilder) createS3(config *ConfigurationBuilder) error {
	var s3Svc s3iface.S3API
	s3Svc = s3.New(bldr.awsSession)
	config.WithService(s3Svc)
	return nil
}

func (bldr *AWSServiceBuilder) createCognito(config *ConfigurationBuilder) error {
	var cognitoSvc cognitoidentityprovideriface.CognitoIdentityProviderAPI
	cognitoSvc = cognitoidentityprovider.New(bldr.awsSession)
	config.WithService(cognitoSvc)
	return nil
}

func (bldr *AWSServiceBuilder) createCodeBuild(config *ConfigurationBuilder) error {
	var codeBuildSvc codebuildiface.CodeBuildAPI
	codeBuildSvc = codebuild.New(bldr.awsSession)
	config.WithService(codeBuildSvc)
	return nil
}
