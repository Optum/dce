package config

import (
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
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
type createrFunc func(config ConfigurationBuilder) error

type AWSServiceBuilder interface {
	WithSTS() *DefaultAWSServiceBuilder
	WithSNS() *DefaultAWSServiceBuilder
	WithSQS() *DefaultAWSServiceBuilder
	WithDynamoDB() *DefaultAWSServiceBuilder
	WithS3() *DefaultAWSServiceBuilder
	WithCognito() *DefaultAWSServiceBuilder
	WithCodeBuild() *DefaultAWSServiceBuilder
	WithSSM() *DefaultAWSServiceBuilder
	Build() (*DefaultConfigurationBuilder, error)
}

// DefaultAWSServiceBuilder is the default implementation of the `DefaultAWSServiceBuilder`
type DefaultAWSServiceBuilder struct {
	handlers   []createrFunc
	awsSession *session.Session
	Config     ConfigurationBuilder
}

// WithSTS tells the builder to add an AWS STS service to the `DefaultConfigurater`
func (bldr *DefaultAWSServiceBuilder) WithSTS() *DefaultAWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createSTS)
	return bldr
}

// WithSNS tells the builder to add an AWS SNS service to the `DefaultConfigurater`
func (bldr *DefaultAWSServiceBuilder) WithSNS() *DefaultAWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createSNS)
	return bldr
}

// WithSQS tells the builder to add an AWS SQS service to the `DefaultConfigurater`
func (bldr *DefaultAWSServiceBuilder) WithSQS() *DefaultAWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createSQS)
	return bldr
}

// WithDynamoDB tells the builder to add an AWS DynamoDB service to the `DefaultConfigurater`
func (bldr *DefaultAWSServiceBuilder) WithDynamoDB() *DefaultAWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createDynamoDB)
	return bldr
}

// WithS3 tells the builder to add an AWS S3 service to the `DefaultConfigurater`
func (bldr *DefaultAWSServiceBuilder) WithS3() *DefaultAWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createS3)
	return bldr
}

// WithCognito tells the builder to add an AWS Cognito service to the `DefaultConfigurater`
func (bldr *DefaultAWSServiceBuilder) WithCognito() *DefaultAWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createCognito)
	return bldr
}

// WithCodeBuild tells the builder to add an AWS CodeBuild service to the `DefaultConfigurater`
func (bldr *DefaultAWSServiceBuilder) WithCodeBuild() *DefaultAWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createCodeBuild)
	return bldr
}

// WithSSM tells the builder to add an AWS SSM service to the `DefaultConfigurater`
func (bldr *DefaultAWSServiceBuilder) WithSSM() *DefaultAWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createSSM)
	return bldr
}

// Build creates and returns a structue with AWS services
func (bldr *DefaultAWSServiceBuilder) Build() (*ConfigurationBuilder, error) {
	err := bldr.Config.Build()
	if err != nil {
		// We failed to build the configuration, so honestly there is no
		// point in continuating...
		return &bldr.Config, AWSConfigurationError(err)
	}

	// Create session is done first, and explicitly, because everything else
	// uses it
	err = bldr.createSession(&bldr.Config)

	if err != nil {
		log.Printf("Could not create session: %s", err.Error())
		return &bldr.Config, AWSConfigurationError(err)
	}

	for _, f := range bldr.handlers {
		err := f(bldr.Config)
		if err != nil {
			log.Printf("Error while trying to execute handler: %s", runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name())
			return &bldr.Config, AWSConfigurationError(err)
		}
	}

	// Setting config values from parameter store requires services to be configured first
	if err := bldr.Config.RetrieveParameterStoreVals(); err != nil {
		return &bldr.Config, AWSConfigurationError(err)
	}

	// make certain build is called before returning.
	return &bldr.Config, nil
}

func (bldr *DefaultAWSServiceBuilder) createSession(config *ConfigurationBuilder) error {
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

func (bldr *DefaultAWSServiceBuilder) createSTS(config ConfigurationBuilder) error {
	var stsSvc stsiface.STSAPI
	stsSvc = sts.New(bldr.awsSession)
	config.WithService(stsSvc)
	return nil
}

func (bldr *DefaultAWSServiceBuilder) createSNS(config ConfigurationBuilder) error {
	var snsSvc snsiface.SNSAPI
	snsSvc = sns.New(bldr.awsSession)
	config.WithService(snsSvc)
	return nil
}

func (bldr *DefaultAWSServiceBuilder) createSQS(config ConfigurationBuilder) error {
	var sqsSvc sqsiface.SQSAPI
	sqsSvc = sqs.New(bldr.awsSession)
	config.WithService(sqsSvc)
	return nil
}

func (bldr *DefaultAWSServiceBuilder) createDynamoDB(config ConfigurationBuilder) error {
	var dynamodbSvc dynamodbiface.DynamoDBAPI
	dynamodbSvc = dynamodb.New(bldr.awsSession)
	config.WithService(dynamodbSvc)
	return nil
}

func (bldr *DefaultAWSServiceBuilder) createS3(config ConfigurationBuilder) error {
	var s3Svc s3iface.S3API
	s3Svc = s3.New(bldr.awsSession)
	config.WithService(s3Svc)
	return nil
}

func (bldr *DefaultAWSServiceBuilder) createCognito(config ConfigurationBuilder) error {
	var cognitoSvc cognitoidentityprovideriface.CognitoIdentityProviderAPI
	cognitoSvc = cognitoidentityprovider.New(bldr.awsSession)
	config.WithService(cognitoSvc)
	return nil
}

func (bldr *DefaultAWSServiceBuilder) createCodeBuild(config ConfigurationBuilder) error {
	var codeBuildSvc codebuildiface.CodeBuildAPI
	codeBuildSvc = codebuild.New(bldr.awsSession)
	config.WithService(codeBuildSvc)
	return nil
}

func (bldr *DefaultAWSServiceBuilder) createSSM(config ConfigurationBuilder) error {
	var SSMSvc ssmiface.SSMAPI
	SSMSvc = ssm.New(bldr.awsSession)
	config.WithService(SSMSvc)
	return nil
}
