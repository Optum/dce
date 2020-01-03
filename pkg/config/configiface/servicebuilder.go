//

package configiface

import "github.com/Optum/dce/pkg/config"

// ServiceBuilder makes working with the ServiceBuild easier
type ServiceBuilder interface {
	// WithSTS tells the builder to add an AWS STS service to the `DefaultConfigurater`
	WithSTS() *config.ServiceBuilder
	// WithSNS tells the builder to add an AWS SNS service to the `DefaultConfigurater`
	WithSNS() *config.ServiceBuilder
	// WithSQS tells the builder to add an AWS SQS service to the `DefaultConfigurater`
	WithSQS() *config.ServiceBuilder
	// WithDynamoDB tells the builder to add an AWS DynamoDB service to the `DefaultConfigurater`
	WithDynamoDB() *config.ServiceBuilder
	// WithS3 tells the builder to add an AWS S3 service to the `DefaultConfigurater`
	WithS3() *config.ServiceBuilder
	// WithCognito tells the builder to add an AWS Cognito service to the `DefaultConfigurater`
	WithCognito() *config.ServiceBuilder
	// WithCodeBuild tells the builder to add an AWS CodeBuild service to the `DefaultConfigurater`
	WithCodeBuild() *config.ServiceBuilder
	// WithSSM tells the builder to add an AWS SSM service to the `DefaultConfigurater`
	WithSSM() *config.ServiceBuilder
	// Build creates and returns a structue with AWS services
	Build() (*config.ConfigurationBuilder, error)
}
