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
	"github.com/aws/aws-sdk-go/service/costexplorer/costexploreriface"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
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

type AwsSession interface {
	client.ConfigProvider
}

type CostExplorerAPI interface {
	costexploreriface.CostExplorerAPI
}
