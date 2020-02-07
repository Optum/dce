package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/account/accountiface/mocks"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/errors"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWhenDelete(t *testing.T) {
	standardHeaders := map[string][]string{
		"Access-Control-Allow-Origin": []string{"*"},
		"Content-Type":                []string{"application/json"},
	}

	tests := []struct {
		name       string
		accountID  string
		expResp    events.APIGatewayProxyResponse
		request    events.APIGatewayProxyRequest
		getAccount *account.Account
		getErr     error
		deleteErr  error
	}{
		{
			name:      "When given good account ID. Then success is returned.",
			accountID: "123456789012",
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusNoContent,
				Body:              "",
				MultiValueHeaders: standardHeaders,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodDelete,
				Path:       "/accounts/123456789012",
			},
			getAccount: &account.Account{
				ID: ptrString("123456789012"),
			},
			getErr: nil,
		},
		{
			name:      "When given bad account ID. Then a not found error is returned.",
			accountID: "210987654321",
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusNotFound,
				Body:              "{\"error\":{\"message\":\"account \\\"210987654321\\\" not found\",\"code\":\"NotFoundError\"}}\n",
				MultiValueHeaders: standardHeaders,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodDelete,
				Path:       "/accounts/210987654321",
			},
			getAccount: nil,
			getErr:     errors.NewNotFound("account", "210987654321"),
		},
		{
			name:      "Given delete failure. Then an error is returned.",
			accountID: "123456789012",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodDelete,
				Path:       "/accounts/123456789012",
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusInternalServerError,
				Body:              "{\"error\":{\"message\":\"failure\",\"code\":\"ServerError\"}}\n",
				MultiValueHeaders: standardHeaders,
			},
			getAccount: &account.Account{
				ID: ptrString("123456789012"),
			},
			getErr:    nil,
			deleteErr: errors.NewInternalServer("failure", nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			accountSvc := mocks.Servicer{}
			accountSvc.On("Get", tt.accountID).Return(
				tt.getAccount, tt.getErr,
			)
			accountSvc.On("Delete", mock.AnythingOfType("*account.Account")).Return(
				tt.deleteErr,
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
