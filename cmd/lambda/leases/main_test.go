package main

import (
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestMain(m *testing.M) {
	os.Setenv("LEASE_ADDED_TOPIC", "mock-lease-added-topic")
	os.Setenv("DECOMMISSION_TOPIC", "mock-decommission-topic")
	os.Setenv("COGNITO_USER_POOL_ID", "mock-cognito-user-pool-id")
	os.Setenv("COGNITO_ROLES_ATTRIBUTE_ADMIN_NAME", "mock-cognito-admin-name")
	os.Setenv("PRINCIPAL_BUDGET_AMOUNT", "1000.00")
	os.Setenv("PRINCIPAL_BUDGET_PERIOD", "Weekly")
	os.Setenv("MAX_LEASE_BUDGET_AMOUNT", "1000.00")
	os.Setenv("MAX_LEASE_PERIOD", "704800")
	os.Setenv("DEFAULT_LEASE_LENGTH_IN_DAYS", "7")
	os.Exit(m.Run())
}

// MockAPIResponse is a helper function to create and return a valid response
// for an API Gateway
func MockAPIResponse(status int, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		MultiValueHeaders: map[string][]string{
			"Content-Type":                []string{"application/json"},
			"Access-Control-Allow-Origin": []string{"*"},
		},
		Body: body,
	}
}

func MockAPIErrorResponse(status int, body string) events.APIGatewayProxyResponse {

	return events.APIGatewayProxyResponse{
		StatusCode: status,
		MultiValueHeaders: map[string][]string{
			"Content-Type":                []string{"application/json"},
			"Access-Control-Allow-Origin": []string{"*"},
		},
		Body: body,
	}
}
