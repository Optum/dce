package main

import (
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestMain(m *testing.M) {
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
