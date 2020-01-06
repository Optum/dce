/*
awsiface package contains interfaces for AWS SDKs.

Wrapping AWS SDK interfaces in our own local interfaces allows
us to generate mocks for them using `mockery`.
Keeping this package separate from other services prevents
cyclical dependencies in generated mock packages.
*/

//go:generate mockery -all
package awsiface

import (
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider/cognitoidentityprovideriface"
	"github.com/aws/aws-sdk-go/service/costexplorer/costexploreriface"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/eventbridge/eventbridgeiface"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

type LambdaAPI interface {
	lambdaiface.LambdaAPI
}

type IAM interface {
	iamiface.IAMAPI
}

type SESAPI interface {
	sesiface.SESAPI
}

type SQSAPI interface {
	sqsiface.SQSAPI
}

type SNSAPI interface {
	snsiface.SNSAPI
}

type EventBridgeAPI interface {
	eventbridgeiface.EventBridgeAPI
}

type S3API interface {
	s3iface.S3API
}

type AwsSession interface {
	client.ConfigProvider
}

type CostExplorerAPI interface {
	costexploreriface.CostExplorerAPI
}

type CognitoIdentityProviderAPI interface {
	cognitoidentityprovideriface.CognitoIdentityProviderAPI
}

type DynamoDBAPI interface {
	dynamodbiface.DynamoDBAPI
}
