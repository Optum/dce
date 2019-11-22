package config

import (
	"fmt"
	"log"
	"reflect"
	"runtime"

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
type createrFunc func(config *DCEConfigBuilder) error

// AWSServiceBuilder is the default implementation of the `AWSServiceBuilder`
type AWSServiceBuilder struct {
	handlers []createrFunc
	Config   *DCEConfigBuilder
}

// WithSession tells the builder to add an AWS session to the result
// func (bldr *DefaultAWSServiceBuilder) WithSession() *DefaultAWSServiceBuilder {
// 	bldr.handlers = append(bldr.handlers, createSession)
// 	return bldr
// }

// WithSTS tells the builder to add an AWS STS service to the `DefaultConfigurater`
func (bldr *AWSServiceBuilder) WithSTS() *AWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, createSTS)
	return bldr
}

// WithSNS tells the builder to add an AWS SNS service to the `DefaultConfigurater`
func (bldr *AWSServiceBuilder) WithSNS() *AWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, createSNS)
	return bldr
}

// WithSQS tells the builder to add an AWS SQS service to the `DefaultConfigurater`
func (bldr *AWSServiceBuilder) WithSQS() *AWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, createSQS)
	return bldr
}

// WithDynamoDB tells the builder to add an AWS DynamoDB service to the `DefaultConfigurater`
func (bldr *AWSServiceBuilder) WithDynamoDB() *AWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, createDynamoDB)
	return bldr
}

// WithS3 tells the builder to add an AWS S3 service to the `DefaultConfigurater`
func (bldr *AWSServiceBuilder) WithS3() *AWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, createS3)
	return bldr
}

// WithCognito tells the builder to add an AWS Cognito service to the `DefaultConfigurater`
func (bldr *AWSServiceBuilder) WithCognito() *AWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, createCognito)
	return bldr
}

// WithCodePipeline tells the builder to add an AWS CodePipeline service to the `DefaultConfigurater`
func (bldr *AWSServiceBuilder) WithCodePipeline() *AWSServiceBuilder {
	bldr.handlers = append(bldr.handlers, createCodePipeline)
	return bldr
}

// Build creates and returns a structue with AWS services
func (bldr *AWSServiceBuilder) Build() (*DCEConfigBuilder, error) {

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

	// make certain build is called before returning.
	err = bldr.Config.Build()
	return bldr.Config, err
}

func createSession(config *DCEConfigBuilder) error {
	awsSession, err := session.NewSession()
	config.WithService(awsSession)
	return err
}

func getSession(config *DCEConfigBuilder) (*session.Session, error) {
	var awsSession *session.Session
	config.GetService(awsSession)
	if awsSession == nil {
		return nil, AWSConfigurationError(fmt.Errorf("error while trying to get session"))
	}
	return awsSession, nil
}

func createSTS(config *DCEConfigBuilder) error {
	if awsSession, err := getSession(config); err == nil {
		var stsSvc stsiface.STSAPI
		stsSvc = sts.New(awsSession)
		config.WithService(stsSvc)
	} else {
		return err
	}
	return nil
}

func createSNS(config *DCEConfigBuilder) error {
	if awsSession, err := getSession(config); err == nil {
		var snsSvc snsiface.SNSAPI
		snsSvc = sns.New(awsSession)
		config.WithService(snsSvc)
	} else {
		return err
	}
	return nil
}

func createSQS(config *DCEConfigBuilder) error {
	if awsSession, err := getSession(config); err == nil {
		var sqsSvc sqsiface.SQSAPI
		sqsSvc = sqs.New(awsSession)
		config.WithService(sqsSvc)
	} else {
		return err
	}
	return nil
}

func createDynamoDB(config *DCEConfigBuilder) error {
	if awsSession, err := getSession(config); err == nil {
		var dynamodbSvc dynamodbiface.DynamoDBAPI
		dynamodbSvc = dynamodb.New(awsSession)
		config.WithService(dynamodbSvc)
	} else {
		return err
	}
	return nil
}

func createS3(config *DCEConfigBuilder) error {
	if awsSession, err := getSession(config); err == nil {
		var s3Svc s3iface.S3API
		s3Svc = s3.New(awsSession)
		config.WithService(s3Svc)
	} else {
		return err
	}
	return nil
}

func createCognito(config *DCEConfigBuilder) error {
	if awsSession, err := getSession(config); err == nil {
		var cognitoSvc cognitoidentityprovideriface.CognitoIdentityProviderAPI
		cognitoSvc = cognitoidentityprovider.New(awsSession)
		config.WithService(cognitoSvc)
	} else {
		return err
	}
	return nil
}

func createCodePipeline(config *DCEConfigBuilder) error {
	if awsSession, err := getSession(config); err == nil {
		var codeBuildSvc codebuildiface.CodeBuildAPI
		codeBuildSvc = codebuild.New(awsSession)
		config.WithService(codeBuildSvc)
	} else {
		return err
	}
	return nil
}
