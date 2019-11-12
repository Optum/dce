package config

import (
	"fmt"
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
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"

	"github.com/Optum/dce/pkg/accountmanager"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/data"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/rolemanager"
)

// AWSSessionKey is the key for the configuration for the AWS session
const AWSSessionKey = "AWSSession"

// ServiceConfigurationError is returned when an AWS service cannot be properly configured.
type ServiceConfigurationError error

// createrFunc internal functions for handling the creation of the services
type createrFunc func(config *ConfigurationBuilder) error

// ServiceBuilder is the default implementation of the `ServiceBuilder`
type ServiceBuilder struct {
	handlers   []createrFunc
	AWSSession *session.Session
	Config     *ConfigurationBuilder
}

// WithSTS tells the builder to add an AWS STS service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithSTS() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createSTS)
	return bldr
}

// WithSNS tells the builder to add an AWS SNS service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithSNS() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createSNS)
	return bldr
}

// WithSQS tells the builder to add an AWS SQS service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithSQS() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createSQS)
	return bldr
}

// WithDynamoDB tells the builder to add an AWS DynamoDB service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithDynamoDB() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createDynamoDB)
	return bldr
}

// WithS3 tells the builder to add an AWS S3 service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithS3() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createS3)
	return bldr
}

// WithCognito tells the builder to add an AWS Cognito service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithCognito() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createCognito)
	return bldr
}

// WithCodeBuild tells the builder to add an AWS CodeBuild service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithCodeBuild() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createCodeBuild)
	return bldr
}

// WithRoleManager tells the builder to add the DCE RoleManager service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithRoleManager() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createRoleManager)
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

// WithAccountManager tells the builder to add Account Manager service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithAccountManager() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createAccountManager)
	return bldr
}

// WithSSM tells the builder to add an AWS SSM service to the `DefaultConfigurater`
func (bldr *ServiceBuilder) WithSSM() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createSSM)
	return bldr
}

// Build creates and returns a structue with AWS services
func (bldr *ServiceBuilder) Build() (*ConfigurationBuilder, error) {

	err := bldr.Config.Build()
	if err != nil {
		// We failed to build the configuration, so honestly there is no
		// point in continuating...
		return bldr.Config, ServiceConfigurationError(err)
	}

	// Create session is done first, and explicitly, because everything else
	// uses it
	err = bldr.createSession(bldr.Config)

	if err != nil {
		log.Printf("Could not create session: %s", err.Error())
		return bldr.Config, ServiceConfigurationError(err)
	}

	for _, f := range bldr.handlers {
		err := f(bldr.Config)
		if err != nil {
			log.Printf("Error while trying to execute handler: %s", runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name())
			return bldr.Config, ServiceConfigurationError(err)
		}
	}

	// make certain build is called before returning.
	return bldr.Config, nil
}

func (bldr *ServiceBuilder) createSession(config *ConfigurationBuilder) error {
	var err error
	region, err := bldr.Config.GetStringVal("AWS_CURRENT_REGION")
	if err == nil {
		log.Printf("Using AWS region \"%s\" to create session...", region)
		bldr.AWSSession, err = session.NewSession(
			&aws.Config{
				Region: aws.String(region),
			},
		)
	} else {
		log.Println("Creating AWS session using defaults...")
		bldr.AWSSession, err = session.NewSession()
	}
	return err
}

func (bldr *ServiceBuilder) createSTS(config *ConfigurationBuilder) error {
	var stsSvc stsiface.STSAPI
	stsSvc = sts.New(bldr.AWSSession)
	config.WithService(stsSvc)
	return nil
}

func (bldr *ServiceBuilder) createSNS(config *ConfigurationBuilder) error {
	var snsSvc snsiface.SNSAPI
	snsSvc = sns.New(bldr.AWSSession)
	config.WithService(snsSvc)
	return nil
}

func (bldr *ServiceBuilder) createSQS(config *ConfigurationBuilder) error {
	var sqsSvc sqsiface.SQSAPI
	sqsSvc = sqs.New(bldr.AWSSession)
	config.WithService(sqsSvc)
	return nil
}

func (bldr *ServiceBuilder) createDynamoDB(config *ConfigurationBuilder) error {
	var dynamodbSvc dynamodbiface.DynamoDBAPI
	dynamodbSvc = dynamodb.New(bldr.AWSSession)
	config.WithService(dynamodbSvc)
	return nil
}

func (bldr *ServiceBuilder) createS3(config *ConfigurationBuilder) error {
	var s3Svc s3iface.S3API
	s3Svc = s3.New(bldr.AWSSession)
	config.WithService(s3Svc)
	return nil
}

func (bldr *ServiceBuilder) createCognito(config *ConfigurationBuilder) error {
	var cognitoSvc cognitoidentityprovideriface.CognitoIdentityProviderAPI
	cognitoSvc = cognitoidentityprovider.New(bldr.AWSSession)
	config.WithService(cognitoSvc)
	return nil
}

func (bldr *ServiceBuilder) createCodeBuild(config *ConfigurationBuilder) error {
	var codeBuildSvc codebuildiface.CodeBuildAPI
	codeBuildSvc = codebuild.New(bldr.AWSSession)
	config.WithService(codeBuildSvc)
	return nil
}

func (bldr *ServiceBuilder) createRoleManager(config *ConfigurationBuilder) error {
	var rmSvc rolemanager.RoleManager
	rmSvc = &rolemanager.IAMRoleManager{}
	config.WithService(rmSvc)
	return nil
}

func (bldr *ServiceBuilder) createDAO(config *ConfigurationBuilder) error {
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

func (bldr *ServiceBuilder) createStorageService(config *ConfigurationBuilder) error {
	var storageService common.Storager
	storageService = &common.S3{}
	config.WithService(storageService)
	return nil
}

func (bldr *ServiceBuilder) createDataService(config *ConfigurationBuilder) error {
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

	dataSvcImpl.AwsDynamoDB = dynamodbSvc
	fmt.Printf("Data Service created: %+v\n", dataSvcImpl)
	config.WithService(dataSvcImpl)
	return nil
}

func (bldr *ServiceBuilder) createAccountManager(config *ConfigurationBuilder) error {

	amSvcImpl := &accountmanager.AccountManager{}

	err := bldr.Config.Unmarshal(amSvcImpl)
	if err != nil {
		return err
	}

	config.WithService(amSvcImpl)
	return nil
}

func (bldr *ServiceBuilder) createSSM(config *ConfigurationBuilder) error {
	var SSMSvc ssmiface.SSMAPI
	SSMSvc = ssm.New(bldr.AWSSession)
	config.WithService(SSMSvc)
	return nil
}
