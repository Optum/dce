package config

import (
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/usage"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"log"
	"reflect"
	"runtime"

	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/data"
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
type createrFunc func() error

// ServiceBuilder is the default implementation of the `ServiceBuild`
type ServiceBuilder struct {
	ConfigurationBuilder
	handlers []createrFunc
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

func (bldr *ServiceBuilder) WithDB() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, func() error {
		awsSession, err := bldr.Session()
		if err != nil {
			return err
		}

		// TODO: Load these env vars from the ServiceBuilder
		// but we need a way for config service definitions to depend
		// on other config fields
		region := common.RequireEnv("AWS_CURRENT_REGION")
		accountTableName := common.RequireEnv("ACCOUNT_DB")
		leaseTableName := common.RequireEnv("LEASE_DB")
		defaultLeaseLength := common.GetEnvInt("DEFAULT_LEASE_LENGTH_IN_DAYS", 7)

		dbSvc := db.New(
			dynamodb.New(
				awsSession,
				aws.NewConfig().WithRegion(region),
			),
			accountTableName,
			leaseTableName,
			defaultLeaseLength,
		)
		bldr.WithService(dbSvc)

		return nil
	})
	return bldr
}

func (bldr *ServiceBuilder) WithTokenService() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, func() error {
		awsSession, err := bldr.Session()
		if err != nil {
			return err
		}

		stsClient := sts.New(awsSession)
		tokenService := &common.STS{
			Client: stsClient,
		}
		bldr.WithService(tokenService)

		return nil
	})
	return bldr
}

func (bldr *ServiceBuilder) WithStorager() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, func() error {
		awsSession, err := bldr.Session()
		if err != nil {
			return err
		}

		bldr.WithService(&common.S3{
			Client:  s3.New(awsSession),
			Manager: s3manager.NewDownloader(awsSession),
		})

		return nil
	})
	return bldr
}

func (bldr *ServiceBuilder) WithNotificationer() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, func() error {
		awsSession, err := bldr.Session()
		if err != nil {
			return err
		}

		bldr.WithService(&common.SNS{
			Client: sns.New(awsSession),
		})

		return nil
	})
	return bldr
}

func (bldr *ServiceBuilder) WithUsageService() *ServiceBuilder {
	bldr.handlers = append(bldr.handlers, func() error {
		var dynDB dynamodb.DynamoDB
		err := bldr.GetService(dynDB)
		if err != nil {
			return err
		}

		bldr.WithService(usage.New(
			&dynDB,
			common.RequireEnv("USAGE_CACHE_DB"),
			"StartDate",
			"PrincipalId",
		))

		return nil
	})
	return bldr
}

// Build creates and returns a structue with AWS services
func (bldr *ServiceBuilder) Build() (*ServiceBuilder, error) {
	err := bldr.ConfigurationBuilder.Build()
	if err != nil {
		// We failed to build the configuration, so honestly there is no
		// point in continuating...
		return bldr, AWSConfigurationError(err)
	}

	// Create session is done first, and explicitly, because everything else
	// uses it
	err = bldr.createSession()

	if err != nil {
		log.Printf("Could not create session: %s", err.Error())
		return bldr, AWSConfigurationError(err)
	}

	for _, f := range bldr.handlers {
		err := f()
		if err != nil {
			log.Printf("Error while trying to execute handler: %s", runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name())
			return bldr, AWSConfigurationError(err)
		}
	}

	// Setting config values from parameter store requires services to be configured first
	if err := bldr.RetrieveParameterStoreVals(); err != nil {
		return bldr, AWSConfigurationError(err)
	}

	// make certain build is called before returning.
	return bldr, nil
}

func (bldr *ServiceBuilder) createSession() error {
	var err error
	region := common.GetEnv("AWS_CURRENT_REGION", "")
	var awsSession *session.Session
	if region == "" {
		log.Printf("Using AWS region \"%s\" to create session...", region)
		awsSession, err = session.NewSession(
			&aws.Config{
				Region: aws.String(region),
			},
		)
		if err != nil {
			return err
		}
	} else {
		log.Println("Creating AWS session using defaults...")
		awsSession, err = session.NewSession()
		if err != nil {
			return err
		}
	}

	bldr.WithService(awsSession)

	return nil
}

func (bldr *ServiceBuilder) Session() (*session.Session, error) {
	var awsSession session.Session
	err := bldr.GetService(&awsSession)
	return &awsSession, err
}

func (bldr *ServiceBuilder) createSTS() error {
	awsSession, err := bldr.Session()
	if err != nil {
		return err
	}
	stsSvc := sts.New(awsSession)
	bldr.WithService(stsSvc)
	return nil
}

func (bldr *ServiceBuilder) createSNS() error {
	awsSession, err := bldr.Session()
	if err != nil {
		return err
	}
	snsSvc := sns.New(awsSession)
	bldr.WithService(snsSvc)
	return nil
}

func (bldr *ServiceBuilder) createSQS() error {
	awsSession, err := bldr.Session()
	if err != nil {
		return err
	}
	sqsSvc := sqs.New(awsSession)
	bldr.WithService(sqsSvc)
	return nil
}

func (bldr *ServiceBuilder) createDynamoDB() error {
	awsSession, err := bldr.Session()
	if err != nil {
		return err
	}
	dynamodbSvc := dynamodb.New(awsSession)
	bldr.WithService(dynamodbSvc)
	return nil
}

func (bldr *ServiceBuilder) createS3() error {
	awsSession, err := bldr.Session()
	if err != nil {
		return err
	}
	s3Svc := s3.New(awsSession)
	bldr.WithService(s3Svc)
	return nil
}

func (bldr *ServiceBuilder) createCognito() error {
	awsSession, err := bldr.Session()
	if err != nil {
		return err
	}
	cognitoSvc := cognitoidentityprovider.New(awsSession)
	bldr.WithService(cognitoSvc)
	return nil
}

func (bldr *ServiceBuilder) createCodeBuild() error {
	awsSession, err := bldr.Session()
	if err != nil {
		return err
	}
	codeBuildSvc := codebuild.New(awsSession)
	bldr.WithService(codeBuildSvc)
	return nil
}

func (bldr *ServiceBuilder) createSSM() error {
	awsSession, err := bldr.Session()
	if err != nil {
		return err
	}
	SSMSvc := ssm.New(awsSession)
	bldr.WithService(SSMSvc)
	return nil
}

func (bldr *ServiceBuilder) createStorageService() error {
	storageService := &common.S3{}
	bldr.WithService(storageService)
	return nil
}

func (bldr *ServiceBuilder) createDataService() error {
	var dynamodbSvc dynamodbiface.DynamoDBAPI
	err := bldr.GetService(&dynamodbSvc)

	if err != nil {
		return err
	}

	dataSvcImpl := &data.Account{}

	err = bldr.Unmarshal(dataSvcImpl)
	if err != nil {
		return err
	}

	dataSvcImpl.DynamoDB = dynamodbSvc

	bldr.WithService(dataSvcImpl)
	return nil
}
