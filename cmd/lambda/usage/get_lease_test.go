package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Optum/dce/pkg/api"
	apiMocks "github.com/Optum/dce/pkg/api/mocks"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/usage"
	"github.com/Optum/dce/pkg/usage/usageiface/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetLease(t *testing.T) {

	tests := []struct {
		name     string
		expResp  events.APIGatewayProxyResponse
		request  events.APIGatewayProxyRequest
		retLease *usage.Lease
		expQuery string
		retErr   error
	}{
		{
			name: "get all lease usage records",
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        200,
				Body:              "{}\n",
				MultiValueHeaders: standardHeaders,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
				Path:       "/usage/lease/lease1/summary",
			},
			expQuery: "lease1",
			retLease: &usage.Lease{},
			retErr:   nil,
		},
		{
			name:     "fail to get accounts",
			expQuery: "lease1",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
				Path:       "/usage/lease/lease1/summary",
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        500,
				Body:              "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
				MultiValueHeaders: standardHeaders,
			},
			retLease: nil,
			retErr:   fmt.Errorf("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			usageSvc := mocks.Servicer{}
			usageSvc.On("GetLease", tt.expQuery).Return(
				tt.retLease, tt.retErr,
			)

			userDetailerSvc := apiMocks.UserDetailer{}
			userDetailerSvc.On("GetUser", mock.AnythingOfType("*events.APIGatewayProxyRequestContext")).Return(&api.User{})

			svcBldr.Config.WithService(&usageSvc).WithService(&userDetailerSvc)
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
