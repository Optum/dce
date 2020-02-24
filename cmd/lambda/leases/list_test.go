package main

import (
	"context"
	"fmt"
	"github.com/Optum/dce/pkg/api"
	apiMocks "github.com/Optum/dce/pkg/api/mocks"
	"github.com/aws/aws-lambda-go/events"
	"net/http"

	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/lease/leaseiface/mocks"
	"github.com/gorilla/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAdminGetLeases(t *testing.T) {

	type response struct {
		StatusCode int
		Body       string
	}
	tests := []struct {
		name            string
		user            *api.User
		expResp         response
		expLink         string
		query           *lease.Lease
		retLeases       *lease.Leases
		retErr          error
		nextAccountID   *string
		nextPrincipalID *string
	}{
		{
			name: "admin gets empty list when no leases",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			query: &lease.Lease{},
			expResp: response{
				StatusCode: 200,
				Body:       "[]\n",
			},
			retLeases: &lease.Leases{},
			retErr:    nil,
		},
		{
			name: "admin can get paged leases belonging to other users",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			query: &lease.Lease{},
			expResp: response{
				StatusCode: 200,
				Body:       "[{\"accountId\":\"123456789012\",\"principalId\":\"User1\"}]\n",
			},
			retLeases: &lease.Leases{
				lease.Lease{
					AccountID:   ptrString("123456789012"),
					PrincipalID: ptrString("User1"),
				},
			},
			nextAccountID:   ptrString("234567890123"),
			nextPrincipalID: ptrString("User2"),
			expLink:         "</leases?limit=1&nextAccountId=234567890123&nextPrincipalId=User2>; rel=\"next\"",
			retErr:          nil,
		},
		{
			name: "user gets empty list when no leases",
			user: &api.User{
				Username: "User1",
				Role:     api.UserGroupName,
			},
			query: &lease.Lease{},
			expResp: response{
				StatusCode: 200,
				Body:       "[]\n",
			},
			retLeases: &lease.Leases{},
			retErr:    nil,
		},
		{
			name: "user can get only their own paged leases",
			user: &api.User{
				Username: "User1",
				Role:     api.UserGroupName,
			},
			query: &lease.Lease{},
			expResp: response{
				StatusCode: 200,
				Body:       "[{\"accountId\":\"123456789012\",\"principalId\":\"User1\"},{\"accountId\":\"133456789012\",\"principalId\":\"User1\"}]\n",
			},
			retLeases: &lease.Leases{
				lease.Lease{
					AccountID:   ptrString("123456789012"),
					PrincipalID: ptrString("User1"),
				},
				lease.Lease{
					AccountID:   ptrString("133456789012"),
					PrincipalID: ptrString("User1"),
				},
			},
			nextAccountID:   ptrString("143456789012"),
			nextPrincipalID: ptrString("User1"),
			expLink:         "</leases?limit=1&nextAccountId=143456789012&nextPrincipalId=User1&principalId=User1>; rel=\"next\"",
			retErr:          nil,
		},
		{
			name: "admin gets 500 when error",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			query: &lease.Lease{
				AccountID:   ptrString("abc123"),
				PrincipalID: ptrString("User2"),
			},
			expResp: response{
				StatusCode: 500,
				Body:       "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
			},
			retLeases: nil,
			retErr:    fmt.Errorf("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://example.com/leases", nil)

			baseRequest = url.URL{}
			baseRequest.Scheme = "https"
			baseRequest.Host = "example.com"
			baseRequest.Path = fmt.Sprintf("%s%s", "unit", "/leases")

			values := url.Values{}
			err := schema.NewEncoder().Encode(tt.query, values)
			assert.Nil(t, err)

			r.URL.RawQuery = values.Encode()

			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			leaseSvc := mocks.Servicer{}

			leaseSvc.On("List", mock.MatchedBy(func(input *lease.Lease) bool {
				if tt.retErr != nil {
					return true
				}
				var authorizationCorrectlyEnforced bool
				if tt.user.Role == api.AdminGroupName {
					// admins own principalID has NOT been added to the query
					authorizationCorrectlyEnforced = input.PrincipalID == nil || *input.PrincipalID != tt.user.Username

				} else {
					// users own principalID has been added to the query
					authorizationCorrectlyEnforced = *input.PrincipalID == tt.user.Username

				}
				accountIDsAreEqual := (input.AccountID != nil && tt.query.AccountID != nil && *input.AccountID == *tt.query.AccountID) || input.AccountID == tt.query.AccountID
				if accountIDsAreEqual && authorizationCorrectlyEnforced {
					if tt.nextAccountID != nil && tt.nextPrincipalID != nil {
						input.NextAccountID = tt.nextAccountID
						input.NextPrincipalID = tt.nextPrincipalID
						input.Limit = ptr64(1)
					}
					return true
				}
				return false
			})).Return(
				tt.retLeases, tt.retErr,
			)

			userDetailSvc := apiMocks.UserDetailer{}
			userDetailSvc.On("GetUser", mock.Anything).Return(tt.user)

			svcBldr.Config.WithService(&userDetailSvc)
			svcBldr.Config.WithService(&leaseSvc)
			_, err = svcBldr.Build()

			assert.Nil(t, err)
			if err == nil {
				Services = svcBldr
			}

			mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/leases"}
			actualResponse, err := Handler(context.TODO(), mockRequest)
			assert.Nil(t, err)
			assert.Equal(t, tt.expResp.StatusCode, actualResponse.StatusCode)
			assert.Equal(t, tt.expResp.Body, actualResponse.Body)
			if tt.expLink != "" {
				assert.Equal(t, tt.expLink, actualResponse.MultiValueHeaders["Link"][0])
			}
		})
	}
}
