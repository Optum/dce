package main

import (
	"context"
	"fmt"
	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/api"
	"net/http"
	"testing"

	accountmocks "github.com/Optum/dce/pkg/account/accountiface/mocks"
	apiMocks "github.com/Optum/dce/pkg/api/mocks"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/lease"
	leasemocks "github.com/Optum/dce/pkg/lease/leaseiface/mocks"
	mockUsage "github.com/Optum/dce/pkg/usage/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWhenCreate(t *testing.T) {
	standardHeaders := map[string][]string{
		"Access-Control-Allow-Origin": []string{"*"},
		"Content-Type":                []string{"application/json"},
	}

	usageSvcMock := &mockUsage.DBer{}
	usageSvcMock.On("GetUsageByPrincipal", mock.Anything, mock.Anything).Return(nil,nil)

	tests := []struct {
		name        string
		user        *api.User
		expResp     events.APIGatewayProxyResponse
		request     events.APIGatewayProxyRequest
		retLease    *lease.Lease
		retAccounts *account.Accounts
		retAccount  *account.Account
		retErr      error
	}{
		{
			name: "When given good values. Then success is returned.",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusCreated,
				Body:              "{}\n",
				MultiValueHeaders: standardHeaders,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Path:       "/leases",
				Body:       "{ \"principalID\": \"User1\", \"adminRoleArn\": \"arn:aws:iam::123456789012:role/AdminRoleArn\" }",
			},
			retAccounts: &account.Accounts{
				account.Account{
					ID:     ptrString("1234567890"),
					Status: account.StatusReady.StatusPtr(),
				},
			},
			retAccount: &account.Account{
				ID:     ptrString("1234567890"),
				Status: account.StatusReady.StatusPtr(),
			},
			retLease: &lease.Lease{},
			retErr:   nil,
		},
		{
			name: "When given bad values. Then a syntax error is returned.",
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusBadRequest,
				Body:              "{\"error\":{\"message\":\"invalid request parameters\",\"code\":\"ClientError\"}}\n",
				MultiValueHeaders: standardHeaders,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Path:       "/leases",
				Body:       "{ \"id: \"123456789012\", \"adminRoleArn\": \"arn:aws:iam::123456789012:role/AdminRoleArn\" }",
			},
			retLease: &lease.Lease{},
			retErr:   nil,
		},
		{
			name: "Given internal failure. Then an internal server error is returned.",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Path:       "/leases",
				Body:       "{ \"id\": \"123456789012\", \"adminRoleArn\": \"arn:aws:iam::123456789012:role/AdminRoleArn\" }",
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusInternalServerError,
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

			leaseSvc := leasemocks.Servicer{}
			accountSvc := accountmocks.Servicer{}

			userDetailSvc := apiMocks.UserDetailer{}
			userDetailSvc.On("GetUser", mock.Anything).Return(tt.user)

			accountSvc.On("List", mock.Anything).Return(
				tt.retAccounts, tt.retErr,
			)
			accountSvc.On("Update", mock.Anything, mock.Anything).Return(
				tt.retAccount, tt.retErr,
			)
			leaseSvc.On("Create", mock.AnythingOfType("*lease.Lease"), mock.Anything).Return(
				tt.retLease, tt.retErr,
			)

			svcBldr.Config.WithService(&accountSvc).WithService(&leaseSvc).WithEnv("PrincipalBudgetPeriod", "PRINCIPAL_BUDGET_PERIOD", "Weekly").WithService(&userDetailSvc)
			_, err := svcBldr.Build()

			assert.Nil(t, err)
			if err == nil {
				Services = svcBldr
			}

			usageSvc = usageSvcMock
			resp, err := Handler(context.TODO(), tt.request)

			assert.Nil(t, err)
			assert.Equal(t, tt.expResp, resp)
		})
	}

}
