package config

import (
	"github.com/Optum/dce/pkg/api"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"log"
	"reflect"
	"runtime"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/account/accountiface"
	"github.com/Optum/dce/pkg/accountmanager"
	"github.com/Optum/dce/pkg/accountmanager/accountmanageriface"
	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/data"
	"github.com/Optum/dce/pkg/data/dataiface"
	"github.com/Optum/dce/pkg/event"
	"github.com/Optum/dce/pkg/event/eventiface"
	"github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/lease/leaseiface"

	"github.com/aws/aws-sdk-go/service/codebuild/codebuildiface"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider/cognitoidentityprovideriface"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"

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

// WithCloudWatchService tells the builder to add an AWS Cognito service to the `DefaultConfigurater`
func (bldr *ServiceBuilder) WithCloudWatchService() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createCloudWatch)
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

// WithLambda tells the builder to add an AWS Lambda service to the `DefaultConfigurater`
func (bldr *ServiceBuilder) WithLambda() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, bldr.createLambda)
	return bldr
}

// WithStorageService tells the builder to add the DCE DAO (DBer) service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithStorageService() *ServiceBuilder {
	bldr.WithS3()
	bldr.handlers = append(bldr.handlers, bldr.createStorageService)
	return bldr
}

// WithAccountDataService tells the builder to add the Data service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithAccountDataService() *ServiceBuilder {
	bldr.WithDynamoDB()
	bldr.handlers = append(bldr.handlers, bldr.createAccountDataService)
	return bldr
}

// WithLeaseDataService tells the builder to add the Data service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithLeaseDataService() *ServiceBuilder {
	bldr.WithDynamoDB()
	bldr.handlers = append(bldr.handlers, bldr.createLeaseDataService)
	return bldr
}

// WithAccountManagerService tells the builder to add the Data service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithAccountManagerService() *ServiceBuilder {
	bldr.WithSTS().WithStorageService()
	bldr.handlers = append(bldr.handlers, bldr.createAccountManagerService)
	return bldr
}

func (bldr *ServiceBuilder) AccountManager() accountmanageriface.Servicer {
	var accountManager accountmanageriface.Servicer
	err := bldr.Config.GetService(&accountManager)
	if err != nil {
		panic(err)
	}
	return accountManager
}

// WithAccountService tells the builder to add the Account service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithAccountService() *ServiceBuilder {
	bldr.WithAccountManagerService().WithEventService().WithAccountDataService()
	bldr.handlers = append(bldr.handlers, bldr.createAccountService)
	return bldr
}

// AccountService returns the account Service for you
func (bldr *ServiceBuilder) AccountService() accountiface.Servicer {

	var accountService accountiface.Servicer
	if err := bldr.Config.GetService(&accountService); err != nil {
		panic(err)
	}

	return accountService
}

// WithLeaseService tells the builder to add the Account service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithLeaseService() *ServiceBuilder {
	// Make sure dependencies are configured
	bldr.
		WithLeaseDataService().
		WithAccountService().
		WithEventService()

	bldr.handlers = append(bldr.handlers, bldr.createLeaseService)
	return bldr
}

// LeaseService returns the lease Service for you
func (bldr *ServiceBuilder) LeaseService() leaseiface.Servicer {

	var leaseSvc leaseiface.Servicer
	if err := bldr.Config.GetService(&leaseSvc); err != nil {
		panic(err)
	}

	return leaseSvc
}

// WithEventService tells the builder to add the Account service to the `ConfigurationBuilder`
func (bldr *ServiceBuilder) WithEventService() *ServiceBuilder {
	bldr.WithSQS().WithSNS()
	bldr.handlers = append(bldr.handlers, bldr.createEventService)
	return bldr
}

func (bldr *ServiceBuilder) WithUserDetailer() *ServiceBuilder {
	bldr.WithCognito()
	bldr.handlers = append(bldr.handlers, bldr.createUserDetailerService)
	return bldr
}

func (bldr *ServiceBuilder) UserDetailer() api.UserDetailer {
	var userDetailer api.UserDetailer
	err := bldr.Config.GetService(&userDetailer)
	if err != nil {
		panic(err)
	}

	return userDetailer
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
	// Don't add the service twice
	var api stsiface.STSAPI
	err := bldr.Config.GetService(&api)
	if err == nil {
		log.Printf("Already added STS service")
		return nil
	}

	stsSvc := sts.New(bldr.awsSession)
	config.WithService(stsSvc)
	return nil
}

func (bldr *ServiceBuilder) createSNS(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var api snsiface.SNSAPI
	err := bldr.Config.GetService(&api)
	if err == nil {
		log.Printf("Already added SNS service")
		return nil
	}
	snsSvc := sns.New(bldr.awsSession)
	config.WithService(snsSvc)
	return nil
}

func (bldr *ServiceBuilder) createSQS(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var api sqsiface.SQSAPI
	err := bldr.Config.GetService(&api)
	if err == nil {
		log.Printf("Already added SQS service")
		return nil
	}
	sqsSvc := sqs.New(bldr.awsSession)
	config.WithService(sqsSvc)
	return nil
}

func (bldr *ServiceBuilder) createDynamoDB(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var api dynamodbiface.DynamoDBAPI
	err := bldr.Config.GetService(&api)
	if err == nil {
		log.Printf("Already added DynamoDB service")
		return nil
	}
	dynamodbSvc := dynamodb.New(bldr.awsSession)
	config.WithService(dynamodbSvc)
	return nil
}

func (bldr *ServiceBuilder) createS3(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var api s3iface.S3API
	err := bldr.Config.GetService(&api)
	if err == nil {
		log.Printf("Already added S3 service")
		return nil
	}
	s3Svc := s3.New(bldr.awsSession)
	config.WithService(s3Svc)
	return nil
}

func (bldr *ServiceBuilder) createCloudWatch(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var api cloudwatch.CloudWatch
	err := bldr.Config.GetService(&api)
	if err == nil {
		log.Printf("Already added CloudWatch service")
		return nil
	}

	cloudWatchSvc := cloudwatch.New(bldr.awsSession)
	config.WithService(cloudWatchSvc)
	return nil
}

func (bldr *ServiceBuilder) createCognito(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var api cognitoidentityprovideriface.CognitoIdentityProviderAPI
	err := bldr.Config.GetService(&api)
	if err == nil {
		log.Printf("Already added Cognito service")
		return nil
	}

	cognitoSvc := cognitoidentityprovider.New(bldr.awsSession)
	config.WithService(cognitoSvc)
	return nil
}

func (bldr *ServiceBuilder) createCodeBuild(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var codeBuildAPI codebuildiface.CodeBuildAPI
	err := bldr.Config.GetService(&codeBuildAPI)
	if err == nil {
		log.Printf("Already added CodeBuild service")
		return nil
	}

	codeBuildSvc := codebuild.New(bldr.awsSession)
	config.WithService(codeBuildSvc)
	return nil
}

func (bldr *ServiceBuilder) createSSM(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var ssmAPI ssmiface.SSMAPI
	err := bldr.Config.GetService(&ssmAPI)
	if err == nil {
		log.Printf("Already added SSM service")
		return nil
	}

	SSMSvc := ssm.New(bldr.awsSession)
	config.WithService(SSMSvc)
	return nil
}

func (bldr *ServiceBuilder) createLambda(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var lambdaAPI lambdaiface.LambdaAPI
	err := bldr.Config.GetService(&lambdaAPI)
	if err == nil {
		log.Printf("Already added Lambda service")
		return nil
	}

	lambdaSvc := lambda.New(bldr.awsSession)
	config.WithService(lambdaSvc)
	return nil
}

func (bldr *ServiceBuilder) createStorageService(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var api common.Storager
	err := bldr.Config.GetService(&api)
	if err == nil {
		log.Printf("Already added Storage service")
		return nil
	}

	storageService := &common.S3{
		Client:  s3.New(bldr.awsSession),
		Manager: s3manager.NewDownloader(bldr.awsSession),
	}

	config.WithService(storageService)
	return nil
}

func (bldr *ServiceBuilder) createUserDetailerService(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var userDetailerAPI api.UserDetailer
	err := bldr.Config.GetService(&userDetailerAPI)
	if err == nil {
		log.Printf("Already added UserDetailer service")
		return nil
	}

	var cognitoSvc cognitoidentityprovider.CognitoIdentityProvider
	err = bldr.Config.GetService(&cognitoSvc)
	if err != nil {
		return err
	}

	userDetailerImpl := &api.UserDetails{}
	err = bldr.Config.Unmarshal(userDetailerImpl)
	if err != nil {
		return err
	}

	userDetailerImpl.CognitoClient = &cognitoSvc

	config.WithService(userDetailerImpl)
	return nil
}

func (bldr *ServiceBuilder) createEventService(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var api eventiface.Servicer
	err := bldr.Config.GetService(&api)
	if err == nil {
		log.Printf("Already added Eventer service")
		return nil
	}

	var sqsService sqsiface.SQSAPI
	err = bldr.Config.GetService(&sqsService)
	if err != nil {
		return err
	}

	var snsService snsiface.SNSAPI
	err = bldr.Config.GetService(&snsService)
	if err != nil {
		return err
	}

	eventSvcInput := event.NewServiceInput{}
	err = bldr.Config.Unmarshal(&eventSvcInput)
	if err != nil {
		return err
	}

	eventSvcInput.SqsClient = sqsService
	eventSvcInput.SnsClient = snsService
	eventSvc, err := event.NewService(eventSvcInput)
	if err != nil {
		return err
	}

	config.WithService(eventSvc)
	return nil
}

func (bldr *ServiceBuilder) createAccountDataService(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var api dataiface.AccountData
	err := bldr.Config.GetService(&api)
	if err == nil {
		log.Printf("Already added Account Data service")
		return nil
	}

	var dynamodbSvc dynamodbiface.DynamoDBAPI
	err = bldr.Config.GetService(&dynamodbSvc)
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

func (bldr *ServiceBuilder) createAccountManagerService(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var api accountmanageriface.Servicer
	err := bldr.Config.GetService(&api)
	if err == nil {
		log.Printf("Already added Account Manager service")
		return nil
	}

	amSvcConfig := accountmanager.ServiceConfig{}
	err = bldr.Config.Unmarshal(&amSvcConfig)
	if err != nil {
		return err
	}

	var stsSvc stsiface.STSAPI
	err = bldr.Config.GetService(&stsSvc)
	if err != nil {
		return err
	}

	var storagerSvc common.Storager
	err = bldr.Config.GetService(&storagerSvc)
	if err != nil {
		return err
	}

	amSvcInput := accountmanager.NewServiceInput{
		Storager: storagerSvc,
		Session:  bldr.awsSession,
		Sts:      stsSvc,
		Config:   amSvcConfig,
	}

	amSvc, err := accountmanager.NewService(amSvcInput)
	if err != nil {
		return err
	}
	config.WithService(amSvc)
	return nil
}

func (bldr *ServiceBuilder) createAccountService(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var api accountiface.Servicer
	err := bldr.Config.GetService(&api)
	if err == nil {
		log.Printf("Already added Account service")
		return nil
	}

	var dataSvc dataiface.AccountData
	err = bldr.Config.GetService(&dataSvc)
	if err != nil {
		return err
	}

	var managerSvc accountmanageriface.Servicer
	err = bldr.Config.GetService(&managerSvc)
	if err != nil {
		return err
	}

	var eventSvc eventiface.Servicer
	err = bldr.Config.GetService(&eventSvc)
	if err != nil {
		return err
	}

	accountSvcInput := account.NewServiceInput{}
	err = bldr.Config.Unmarshal(&accountSvcInput)
	if err != nil {
		return err
	}

	accountSvcInput.DataSvc = dataSvc
	accountSvcInput.ManagerSvc = managerSvc
	accountSvcInput.EventSvc = eventSvc

	accountSvc := account.NewService(accountSvcInput)

	config.WithService(accountSvc)
	return nil
}

func (bldr *ServiceBuilder) createLeaseDataService(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var api dataiface.LeaseData
	err := bldr.Config.GetService(&api)
	if err == nil {
		log.Printf("Already added Lease Data service")
		return nil
	}

	var dynamodbSvc dynamodbiface.DynamoDBAPI
	err = bldr.Config.GetService(&dynamodbSvc)

	if err != nil {
		return err
	}

	dataSvcImpl := &data.Lease{}

	err = bldr.Config.Unmarshal(dataSvcImpl)
	if err != nil {
		return err
	}

	dataSvcImpl.DynamoDB = dynamodbSvc

	config.WithService(dataSvcImpl)
	return nil
}

func (bldr *ServiceBuilder) createLeaseService(config ConfigurationServiceBuilder) error {
	// Don't add the service twice
	var api leaseiface.Servicer
	err := bldr.Config.GetService(&api)
	if err == nil {
		log.Printf("Already added Lease service")
		return nil
	}

	var dataSvc dataiface.LeaseData
	err = bldr.Config.GetService(&dataSvc)
	if err != nil {
		return err
	}

	var accountSvc lease.AccountServicer
	err = bldr.Config.GetService(&accountSvc)
	if err != nil {
		return err
	}

	leaseSvc := lease.NewService(
		lease.NewServiceInput{
			DataSvc: dataSvc,
			AccountSvc: accountSvc,
		},
	)

	config.WithService(leaseSvc)
	return nil
}
