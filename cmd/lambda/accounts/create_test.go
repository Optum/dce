package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/account/accountiface/mocks"
	"github.com/Optum/dce/pkg/config"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWhenCreate(t *testing.T) {
	standardMultiValueHeaders := map[string][]string{
		"Access-Control-Allow-Origin": []string{"*"},
		"Content-Type":                []string{"application/json"},
	}
	standardHeaders := map[string]string{
		"Access-Control-Allow-Origin": "*",
		"Content-Type":                "application/json",
	}

	tests := []struct {
		name       string
		expResp    events.APIGatewayProxyResponse
		request    events.APIGatewayProxyRequest
		retAccount *account.Account
		retErr     error
	}{
		{
			name: "When given good values. Then success is returned.",
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusCreated,
				Body:              "{}\n",
				Headers:           standardHeaders,
				MultiValueHeaders: standardMultiValueHeaders,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Path:       "/accounts",
				Body:       "{ \"id\": \"123456789012\", \"adminRoleArn\": \"arn:aws:iam::123456789012:role/AdminRoleArn\" }",
			},
			retAccount: &account.Account{},
			retErr:     nil,
		},
		{
			name: "When given bad values. Then a syntax error is returned.",
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusBadRequest,
				Body:              "{\"error\":{\"message\":\"invalid request parameters\",\"code\":\"ClientError\"}}\n",
				Headers:           standardHeaders,
				MultiValueHeaders: standardMultiValueHeaders,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Path:       "/accounts",
				Body:       "{ \"id: \"123456789012\", \"adminRoleArn\": \"arn:aws:iam::123456789012:role/AdminRoleArn\" }",
			},
			retAccount: &account.Account{},
			retErr:     nil,
		},
		{
			name: "Given internal failure. Then an internal server error is returned.",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Path:       "/accounts",
				Body:       "{ \"id\": \"123456789012\", \"adminRoleArn\": \"arn:aws:iam::123456789012:role/AdminRoleArn\" }",
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusInternalServerError,
				Body:              "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
				Headers:           standardHeaders,
				MultiValueHeaders: standardMultiValueHeaders,
			},
			retAccount: nil,
			retErr:     fmt.Errorf("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			accountSvc := mocks.Servicer{}
			accountSvc.On("Create", mock.AnythingOfType("*account.Account")).Return(
				tt.retAccount, tt.retErr,
			)
			svcBldr.Config.WithService(&accountSvc)
			_, err := svcBldr.Build()

			assert.Nil(t, err)
			if err == nil {
				Services = svcBldr
			}

			resp, err := Handler(context.TODO(), tt.request)

			assert.Nil(t, err)
			assert.Equal(t, tt.expResp, resp)
		})
	}

}
