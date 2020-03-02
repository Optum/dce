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

func TestListLease(t *testing.T) {

	type pagination struct {
		NextDate        *int64
		NextPrincipalID *string
		NextLeaseID     *string
	}

	tests := []struct {
		name      string
		expResp   events.APIGatewayProxyResponse
		request   events.APIGatewayProxyRequest
		query     *usage.Lease
		retLeases *usage.Leases
		retErr    error
		next      *pagination
	}{
		{
			name:  "get all lease usage records",
			query: &usage.Lease{},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        200,
				Body:              "[]\n",
				MultiValueHeaders: standardHeaders,
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
				Path:       "/usage/lease",
			},
			retLeases: &usage.Leases{},
			retErr:    nil,
		},
		{
			name:  "get paged leases",
			query: &usage.Lease{},
			expResp: events.APIGatewayProxyResponse{
				StatusCode: 200,
				Body:       "[{\"principalId\":\"principal\"}]\n",
				MultiValueHeaders: map[string][]string{
					"Access-Control-Allow-Origin": []string{"*"},
					"Content-Type":                []string{"application/json"},
					"Link":                        []string{"</usage/lease?nextDate=11111&nextLeaseId=lease2&nextPrincipalId=user2>; rel=\"next\""},
				},
			},
			retLeases: &usage.Leases{
				usage.Lease{
					PrincipalID: ptrString("principal"),
				},
			},
			next: &pagination{
				NextPrincipalID: ptrString("user2"),
				NextLeaseID:     ptrString("lease2"),
				NextDate:        ptr64(11111),
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
				Path:       "/usage/lease",
			},
			retErr: nil,
		},
		{
			name: "fail to get accounts",
			query: &usage.Lease{
				PrincipalID: ptrString("user"),
			},
			request: events.APIGatewayProxyRequest{
				HTTPMethod: http.MethodGet,
				Path:       "/usage/lease?principalId=user",
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode:        500,
				Body:              "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
				MultiValueHeaders: standardHeaders,
			},
			retLeases: nil,
			retErr:    fmt.Errorf("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			usageSvc := mocks.Servicer{}
			usageSvc.On("ListLease", mock.MatchedBy(func(input *usage.Lease) bool {
				if (input.PrincipalID != nil && tt.query.PrincipalID != nil && *input.PrincipalID == *tt.query.PrincipalID) || input.PrincipalID == tt.query.PrincipalID {
					if tt.next != nil {
						input.NextPrincipalID = tt.next.NextPrincipalID
						input.NextLeaseID = tt.next.NextLeaseID
						input.NextDate = tt.next.NextDate
					}
					return true
				}
				return false
			})).Return(
				tt.retLeases, tt.retErr,
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
