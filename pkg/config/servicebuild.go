package config

import (
	"log"
	"reflect"
	"runtime"

	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/data"
	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/ssm"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sts"
)

// AWSSessionKey is the key for the configuration for the AWS session
const AWSSessionKey = "AWSSession"

// AWSConfigurationError is returned when an AWS service cannot be properly configured.
type AWSConfigurationError error

// ConfigurationBuildBuilder Interface for adding the build function
type ConfigurationBuildBuilder interface {
	Build() error
}

// ConfigurationServiceBuilder Interface for adding services to the config
type ConfigurationServiceBuilder interface {
	ConfigurationBuildBuilder
	WithService(svc interface{}) *ConfigurationBuilder
}

// createrFunc internal functions for handling the creation of the services
type createrFunc func(config ConfigurationServiceBuilder) error

// ServiceBuilder is the default implementation of the `ServiceBuild`
type ServiceBuilder struct {
	handlers   []createrFunc
	awsSession *session.Session
	Config     *ConfigurationBuilder
}

// WithSTS tells the builder to add an AWS STS service to the `DefaultConfigurater`
func (bldr *ServiceBuilder) WithSTS() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createSTS)
	return bldr
}

// WithSNS tells the builder to add an AWS SNS service to the `DefaultConfigurater`
func (bldr *ServiceBuilder) WithSNS() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createSNS)
	return bldr
}

// WithSQS tells the builder to add an AWS SQS service to the `DefaultConfigurater`
func (bldr *ServiceBuilder) WithSQS() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createSQS)
	return bldr
}

// WithDynamoDB tells the builder to add an AWS DynamoDB service to the `DefaultConfigurater`
func (bldr *ServiceBuilder) WithDynamoDB() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createDynamoDB)
	return bldr
}

// WithS3 tells the builder to add an AWS S3 service to the `DefaultConfigurater`
func (bldr *ServiceBuilder) WithS3() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createS3)
	return bldr
}

// WithCognito tells the builder to add an AWS Cognito service to the `DefaultConfigurater`
func (bldr *ServiceBuilder) WithCognito() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createCognito)
	return bldr
}

// WithCodeBuild tells the builder to add an AWS CodeBuild service to the `DefaultConfigurater`
func (bldr *ServiceBuilder) WithCodeBuild() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createCodeBuild)
	return bldr
}

// WithSSM tells the builder to add an AWS SSM service to the `DefaultConfigurater`
func (bldr *ServiceBuilder) WithSSM() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createSSM)
	return bldr
}

// WithDAO tells the builder to add the DCE DAO (DBer) service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithDAO() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createDAO)
	return bldr
}

// WithStorageService tells the builder to add the DCE DAO (DBer) service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithStorageService() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createStorageService)
	return bldr
}

// WithDataService tells the builder to add the Data service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithDataService() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createDataService)
	return bldr
}

// Build creates and returns a structue with AWS services
func (bldr *ServiceBuilder) Build() (*ConfigurationBuilder, error) {
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

	// Setting config values from parameter store requires services to be configured first
	if err := bldr.Config.RetrieveParameterStoreVals(); err != nil {
		return bldr.Config, AWSConfigurationError(err)
	}

	// make certain build is called before returning.
	return bldr.Config, nil
}

func (bldr *ServiceBuilder) createSession(config ConfigurationServiceBuilder) error {
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

func (bldr *ServiceBuilder) createSTS(config ConfigurationServiceBuilder) error {
	stsSvc := sts.New(bldr.awsSession)
	config.WithService(stsSvc)
	return nil
}

func (bldr *ServiceBuilder) createSNS(config ConfigurationServiceBuilder) error {
	snsSvc := sns.New(bldr.awsSession)
	config.WithService(snsSvc)
	return nil
}

func (bldr *ServiceBuilder) createSQS(config ConfigurationServiceBuilder) error {
	sqsSvc := sqs.New(bldr.awsSession)
	config.WithService(sqsSvc)
	return nil
}

func (bldr *ServiceBuilder) createDynamoDB(config ConfigurationServiceBuilder) error {
	dynamodbSvc := dynamodb.New(bldr.awsSession)
	config.WithService(dynamodbSvc)
	return nil
}

func (bldr *ServiceBuilder) createS3(config ConfigurationServiceBuilder) error {
	s3Svc := s3.New(bldr.awsSession)
	config.WithService(s3Svc)
	return nil
}

func (bldr *ServiceBuilder) createCognito(config ConfigurationServiceBuilder) error {
	cognitoSvc := cognitoidentityprovider.New(bldr.awsSession)
	config.WithService(cognitoSvc)
	return nil
}

func (bldr *ServiceBuilder) createCodeBuild(config ConfigurationServiceBuilder) error {
	codeBuildSvc := codebuild.New(bldr.awsSession)
	config.WithService(codeBuildSvc)
	return nil
}

func (bldr *ServiceBuilder) createSSM(config ConfigurationServiceBuilder) error {
	SSMSvc := ssm.New(bldr.awsSession)
	config.WithService(SSMSvc)
	return nil
}

func (bldr *ServiceBuilder) createDAO(config ConfigurationServiceBuilder) error {
	var daoSvc db.DBer

	var dynamodbSvc dynamodbiface.DynamoDBAPI
	err := bldr.Config.GetService(&dynamodbSvc)

	if err != nil {
		log.Println("Could not find DynamoDB service. Call WithDynamoDB() before WithDAO()")
		return err
	}

	daoSvcImpl := db.DB{}

	err = bldr.Config.Unmarshal(&daoSvcImpl)

	if err != nil {
		log.Printf("Error while trying to create DB from env: %s", err.Error())
		return err
	}

	daoSvcImpl.Client = dynamodbSvc

	daoSvc = &daoSvcImpl

	config.WithService(daoSvc)
	return nil
}

func (bldr *ServiceBuilder) createStorageService(config ConfigurationServiceBuilder) error {
	storageService := &common.S3{}
	config.WithService(storageService)
	return nil
}

func (bldr *ServiceBuilder) createDataService(config ConfigurationServiceBuilder) error {
	var dynamodbSvc dynamodbiface.DynamoDBAPI
	err := bldr.Config.GetService(&dynamodbSvc)

	if err != nil {
		return err
	}

	dataSvcImpl := &data.Account{}

	err = bldr.Config.Unmarshal(dataSvcImpl)
	if err != nil {
		return err
	}

	dataSvcImpl.DynamoDB = dynamodbSvc

	config.WithService(dataSvcImpl)
	return nil
}
