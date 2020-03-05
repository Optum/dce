package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/mitchellh/mapstructure"
)

var (
	testConfig configuration
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
}

func setup() {
	t := &testing.T{}
	tfOpts := &terraform.Options{
		TerraformDir: "../../modules",
	}
	tfOut := terraform.OutputAll(t, tfOpts)

	_ = mapstructure.Decode(tfOut, &testConfig)
	TruncateTables(t)
}

func cleanup() {
	t := &testing.T{}
	TruncateTables(t)
}

func TestMain(m *testing.M) {
	setup()
	fmt.Printf("%+v", testConfig)
	code := m.Run()
	cleanup()
	os.Exit(code)
}
