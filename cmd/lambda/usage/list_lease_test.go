package main

import (
	"context"
	"fmt"
	"testing"

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
		expLink   string
		query     *usage.Lease
		retLeases *usage.Leases
		retErr    error
		next      pagination
	}{
		{
			name:  "get all lease usage records",
			query: &usage.Lease{},
			expResp: events.APIGatewayProxyResponse{
				StatusCode: 200,
				Body:       "[]\n",
			},
			retLeases: &usage.Leases{},
			retErr:    nil,
		},
		{
			name:  "get paged leases",
			query: &usage.Lease{},
			expResp: events.APIGatewayProxyResponse{
				StatusCode: 200,
				Body:       "[{\"principalId\":\"test\"}]\n",
			},
			retLeases: &usage.Leases{
				usage.Lease{
					PrincipalID: ptrString("principal"),
				},
			},
			expLink: "<https://example.com/unit/accounts?limit=1&nextId=234567890123>; rel=\"next\"",
			retErr:  nil,
		},
		{
			name: "fail to get accounts",
			query: &usage.Lease{
				PrincipalID: ptrString("user"),
			},
			expResp: events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
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
			usageSvc.On("ListLease", mock.AnythingOfType("*usage.Lease")).Return(
				tt.retLeases, tt.retErr,
			)
			svcBldr.Config.WithService(&usageSvc)
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
