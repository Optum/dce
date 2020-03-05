package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codebuild"
	"github.com/aws/aws-sdk-go/service/codebuild/codebuildiface"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"
)

var (
	testConfig         configuration
	dbSvc              *db.DB
	sqsSvc             sqsiface.SQSAPI
	codeBuildSvc       codebuildiface.CodeBuildAPI
	dynamoDbSvc        dynamodbiface.DynamoDBAPI
	sqsResetURL        string
	codeBuildResetName string
)

type configuration struct {
	AwsRegion               string `mapstructure:"aws_region"`
	ApiUrl                  string `mapstructure:"api_url"`
	AccountTable            string `mapstructure:"accounts_table_name"`
	LeaseTable              string `mapstructure:"leases_table_name"`
	PrincipalTable          string `mapstructure:"principal_table_name"`
	ApiPolicyAccessArn      string `mapstructure:"api_access_policy_arn"`
	RolePolicyArn           string `mapstructure:"role_user_policy"`
	PrincipalRoleName       string `mapstructure:"principal_role_name"`
	CognitoUserPoolId       string `mapstructure:"cognito_user_pool_id"`
	CognitoUserPoolClientId string `mapstructure:"cognito_user_pool_client_id"`
	CognitoUserPoolEndpoint string `mapstructure:"cognito_user_pool_endpoint"`
	CognitoIdentityPoolId   string `mapstructure:"cognito_identity_pool_id"`
	SfnLeaseUsageArn        string `mapstructure:"sfn_lease_usage_arn"`
	SqsUrl                  string `mapstructure:"sqs_reset_queue_url"`
	CodeBuildProject        string `mapstructure:"codebuild_reset_name"`
}

func setup() {
	t := &testing.T{}
	tfOpts := &terraform.Options{
		TerraformDir: "../../modules",
	}
	tfOut := terraform.OutputAll(t, tfOpts)

	_ = mapstructure.Decode(tfOut, &testConfig)

	// Configure the DB service
	awsSession, err := session.NewSession()
	require.Nil(t, err)
	dbSvc = db.New(
		dynamodb.New(
			awsSession,
			aws.NewConfig().WithRegion(testConfig.AwsRegion),
		),
		testConfig.AccountTable,
		testConfig.LeaseTable,
		7,
	)
	dbSvc.ConsistentRead = true

	dynamoDbSvc = dynamodb.New(
		awsSession,
		aws.NewConfig().WithRegion(tfOut["aws_region"].(string)),
	)

	sqsSvc = sqs.New(
		awsSession,
		aws.NewConfig().WithRegion(tfOut["aws_region"].(string)),
	)
	codeBuildSvc = codebuild.New(
		awsSession,
		aws.NewConfig().WithRegion(tfOut["aws_region"].(string)),
	)
}

func cleanup() {
}

func TestMain(m *testing.M) {
	setup()
	fmt.Printf("%+v", testConfig)
	code := m.Run()
	cleanup()
	os.Exit(code)
}
