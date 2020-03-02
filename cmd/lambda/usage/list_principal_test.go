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

func TestListPrincipal(t *testing.T) {

	type pagination struct {
		NextDate        *int64
		NextPrincipalID *string
	}

	tests := []struct {
		name          string
		expResp       events.APIGatewayProxyResponse
		request       events.APIGatewayProxyRequest
		query         *usage.Principal
		retPrincipals *usage.Principals
		retErr        error
		next          *pagination
	}{
		{
			name:  "get all lease usage records",
			query: &usage.Principal{},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        200,
				Body:              "[]\n",
				MultiValueHeaders: standardHeaders,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
				Path:       "/usage/principal",
			},
			retPrincipals: &usage.Principals{},
			retErr:        nil,
		},
		{
			name:  "get paged leases",
			query: &usage.Principal{},
			expResp: events.APIGatewayProxyResponse{
				StatusCode: 200,
				Body:       "[{\"principalId\":\"principal\"}]\n",
				MultiValueHeaders: map[string][]string{
					"Access-Control-Allow-Origin": []string{"*"},
					"Content-Type":                []string{"application/json"},
					"Link":                        []string{"</usage/principal?nextDate=11111&nextPrincipalId=user2>; rel=\"next\""},
				},
			},
			retPrincipals: &usage.Principals{
				usage.Principal{
					PrincipalID: ptrString("principal"),
				},
			},
			next: &pagination{
				NextPrincipalID: ptrString("user2"),
				NextDate:        ptr64(11111),
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
				Path:       "/usage/principal",
			},
			retErr: nil,
		},
		{
			name: "fail to get accounts",
			query: &usage.Principal{
				PrincipalID: ptrString("user"),
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
				Path:       "/usage/principal?principalId=user",
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        500,
				Body:              "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
				MultiValueHeaders: standardHeaders,
			},
			retPrincipals: nil,
			retErr:        fmt.Errorf("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			usageSvc := mocks.Servicer{}
			usageSvc.On("ListPrincipal", mock.MatchedBy(func(input *usage.Principal) bool {
				if (input.PrincipalID != nil && tt.query.PrincipalID != nil && *input.PrincipalID == *tt.query.PrincipalID) || input.PrincipalID == tt.query.PrincipalID {
					if tt.next != nil {
						input.NextPrincipalID = tt.next.NextPrincipalID
						input.NextDate = tt.next.NextDate
					}
					return true
				}
				return false
			})).Return(
				tt.retPrincipals, tt.retErr,
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

func TestListForPrincipal(t *testing.T) {

	tests := []struct {
		name     string
		expResp  events.APIGatewayProxyResponse
		request  events.APIGatewayProxyRequest
		retLease *usage.Principals
		expQuery *usage.Principal
		retErr   error
	}{
		{
			name: "get all principal usage records for principal",
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        200,
				Body:              "[]\n",
				MultiValueHeaders: standardHeaders,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
				Path:       "/usage/principal/user1",
			},
			expQuery: &usage.Principal{
				PrincipalID: ptrString("user1"),
			},
			retLease: &usage.Principals{},
			retErr:   nil,
		},
		{
			name: "fail to get principal usage records",
			expQuery: &usage.Principal{
				PrincipalID: ptrString("user1"),
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
				Path:       "/usage/principal/user1",
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
			usageSvc.On("ListPrincipal", tt.expQuery).Return(tt.retLease, tt.retErr)

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
