package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/api"

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

func TestWhenCreateSuccess(t *testing.T) {
	standardHeaders := map[string][]string{
		"Access-Control-Allow-Origin": []string{"*"},
		"Content-Type":                []string{"application/json"},
	}

	usageSvcMock := &mockUsage.DBer{}
	usageSvcMock.On("GetUsageByPrincipal", mock.Anything, mock.Anything).Return(nil, nil)

	tests := []struct {
		name         string
		user         *api.User
		expResp      events.APIGatewayProxyResponse
		request      events.APIGatewayProxyRequest
		retLease     *lease.Lease
		retAccounts  *account.Accounts
		retAccount   *account.Account
		retListErr   error
		retUpdateErr error
		retCreateErr error
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
				Body:       "{ \"principalId\": \"User1\", \"budgetAmount\": 200.00 }",
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
			retLease:     &lease.Lease{},
			retListErr:   nil,
			retUpdateErr: nil,
			retCreateErr: nil,
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
				tt.retAccounts, tt.retListErr,
			)
			accountSvc.On("Update", mock.Anything, mock.Anything).Return(
				tt.retAccount, tt.retUpdateErr,
			)
			leaseSvc.On("Create", mock.AnythingOfType("*lease.Lease"), mock.Anything).Return(
				tt.retLease, tt.retCreateErr,
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

func TestWhenCreateError(t *testing.T) {
	standardHeaders := map[string][]string{
		"Access-Control-Allow-Origin": []string{"*"},
		"Content-Type":                []string{"application/json"},
	}

	usageSvcMock := &mockUsage.DBer{}
	usageSvcMock.On("GetUsageByPrincipal", mock.Anything, mock.Anything).Return(nil, fmt.Errorf("Error"))

	tests := []struct {
		name         string
		user         *api.User
		expResp      events.APIGatewayProxyResponse
		request      events.APIGatewayProxyRequest
		retLease     *lease.Lease
		retAccounts  *account.Accounts
		retAccount   *account.Account
		retListErr   error
		retUpdateErr error
		retCreateErr error
	}{
		{
			name: "When principalId is missing. Then a client error is returned.",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Path:       "/leases",
				Body:       "{\"budgetAmount\": 200.00 }",
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusBadRequest,
				Body:              "{\"error\":{\"message\":\"invalid request parameters: missing principalId\",\"code\":\"ClientError\"}}\n",
				MultiValueHeaders: standardHeaders,
			},
			retLease:     nil,
			retListErr:   nil,
			retUpdateErr: nil,
			retCreateErr: nil,
		},
		{
			name: "When given bad values like budget amount is a string. Then a syntax error is returned.",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Path:       "/leases",
				Body:       "{ \"principalId\": \"User1\", \"budgetAmount\": \"200.00\", }",
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusBadRequest,
				Body:              "{\"error\":{\"message\":\"invalid request parameters\",\"code\":\"ClientError\"}}\n",
				MultiValueHeaders: standardHeaders,
			},
			retAccounts: &account.Accounts{
				account.Account{
					ID:     ptrString("1234567890"),
					Status: account.StatusReady.StatusPtr(),
				},
			},
			retLease:     &lease.Lease{},
			retListErr:   nil,
			retUpdateErr: nil,
			retCreateErr: nil,
		},
		{
			name: "When non admin makes creates lease request. Then an unauthorized error is returned.",
			user: &api.User{
				Username: "admin1",
				Role:     api.UserGroupName,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Path:       "/leases",
				Body:       "{ \"principalId\": \"User1\", \"budgetAmount\": 200.00 }",
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusUnauthorized,
				Body:              "{\"error\":{\"message\":\"User [admin1] with role: [User] attempted to act on a lease for [User1], but was not authorized\",\"code\":\"UnauthorizedError\"}}\n",
				MultiValueHeaders: standardHeaders,
			},
			retAccounts: &account.Accounts{
				account.Account{
					ID:     ptrString("1234567890"),
					Status: account.StatusReady.StatusPtr(),
				},
			},
			retLease:     &lease.Lease{},
			retListErr:   nil,
			retUpdateErr: nil,
			retCreateErr: nil,
		},
		{
			name: "When checking for first available ready account fails. Then an internal server error is returned.",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Path:       "/leases",
				Body:       "{ \"principalId\": \"User1\", \"budgetAmount\": 200.00 }",
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusInternalServerError,
				Body:              "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
				MultiValueHeaders: standardHeaders,
			},
			retLease:     nil,
			retListErr:   fmt.Errorf("failure"),
			retUpdateErr: nil,
			retCreateErr: nil,
		},
		{
			name: "When no available accounts to lease. Then an internal server error is returned.",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Path:       "/leases",
				Body:       "{ \"principalId\": \"User1\", \"budgetAmount\": 200.00 }",
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusInternalServerError,
				Body:              "{\"error\":{\"message\":\"No Available accounts at this moment\",\"code\":\"ServerError\"}}\n",
				MultiValueHeaders: standardHeaders,
			},
			retAccounts:  &account.Accounts{},
			retLease:     nil,
			retListErr:   nil,
			retUpdateErr: nil,
			retCreateErr: nil,
		},
		{
			name: "When updating account status to leased fails. Then an internal server error is returned.",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Path:       "/leases",
				Body:       "{ \"principalId\": \"User1\", \"budgetAmount\": 200.00 }",
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusInternalServerError,
				Body:              "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
				MultiValueHeaders: standardHeaders,
			},
			retAccounts: &account.Accounts{
				account.Account{
					ID:     ptrString("1234567890"),
					Status: account.StatusReady.StatusPtr(),
				},
			},
			retLease:     nil,
			retListErr:   nil,
			retUpdateErr: fmt.Errorf("failure"),
			retCreateErr: nil,
		},
		{
			name: "When getting principal spend fails. Then an internal server error is returned.",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Path:       "/leases",
				Body:       "{ \"principalId\": \"User1\", \"budgetAmount\": 200.00 }",
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusInternalServerError,
				Body:              "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
				MultiValueHeaders: standardHeaders,
			},
			retAccounts: &account.Accounts{
				account.Account{
					ID:     ptrString("1234567890"),
					Status: account.StatusReady.StatusPtr(),
				},
			},
			retLease:     nil,
			retListErr:   nil,
			retUpdateErr: nil,
			retCreateErr: nil,
		},
		{
			name: "When creating lease fails. Then an internal server error is returned.",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodPost,
				Path:       "/leases",
				Body:       "{ \"principalId\": \"User1\", \"budgetAmount\": 200.00 }",
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        http.StatusInternalServerError,
				Body:              "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
				MultiValueHeaders: standardHeaders,
			},
			retAccounts: &account.Accounts{
				account.Account{
					ID:     ptrString("1234567890"),
					Status: account.StatusReady.StatusPtr(),
				},
			},
			retLease:     nil,
			retListErr:   nil,
			retUpdateErr: nil,
			retCreateErr: fmt.Errorf("Error"),
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
				tt.retAccounts, tt.retListErr,
			)
			accountSvc.On("Update", mock.Anything, mock.Anything).Return(
				tt.retAccount, tt.retUpdateErr,
			)
			leaseSvc.On("Create", mock.AnythingOfType("*lease.Lease"), mock.Anything).Return(
				tt.retLease, tt.retCreateErr,
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
