package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Optum/Redbox/pkg/api/response"
	"github.com/aws/aws-lambda-go/events"
)

func TestMain(m *testing.M) {
	os.Setenv("ACCOUNT_CREATED_TOPIC_ARN", "mock-account-created-topic")
	os.Setenv("PRINCIPAL_ROLE_NAME", "RedboxPrincipal")
	os.Setenv("RESET_SQS_URL", "mock.queue.url")
	os.Setenv("PRINCIPAL_MAX_SESSION_DURATION", "100")
	os.Setenv("PRINCIPAL_POLICY_NAME", "RedboxPrincipalDefaultPolicy")
	os.Setenv("PRINCIPAL_IAM_DENY_TAGS", "Redbox,CantTouchThis")
	os.Setenv("ACCOUNT_DELETED_TOPIC_ARN", "test:arn")
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

func MockAPIErrorResponse(status int, errorCode string, message string) events.APIGatewayProxyResponse {

	errorJSON, _ := json.Marshal(response.CreateErrorResponse(errorCode, message))

	return events.APIGatewayProxyResponse{
		StatusCode: status,
		MultiValueHeaders: map[string][]string{
			"Content-Type":                []string{"application/json"},
			"Access-Control-Allow-Origin": []string{"*"},
		},
		Body: string(errorJSON),
	}
}
