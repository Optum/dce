package main

import (
	"github.com/Optum/dce/pkg/api"
	apiMocks "github.com/Optum/dce/pkg/api/mocks"
	"github.com/stretchr/testify/mock"

	"testing"

	gErrors "errors"
	"fmt"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/lease/leaseiface/mocks"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http/httptest"

	"context"
	"github.com/aws/aws-lambda-go/events"
	"net/http"
)

func TestGetLeaseByID(t *testing.T) {

	type response struct {
		StatusCode int
		Body       string
	}
	tests := []struct {
		name     string
		user     *api.User
		expResp  response
		leaseID  string
		retLease *lease.Lease
		retErr   error
	}{
		{
			name:    "When admin Get lease for other user service returns a success",
			user:  &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			leaseID: "abc123",
			expResp: response{
				StatusCode: 200,
				Body:       "{\"principalId\":\"user1\"}\n",
			},
			retLease: &lease.Lease{
				PrincipalID:              ptrString("user1"),
			},
			retErr:   nil,
		},
		{
			name:    "When user Get lease for self service returns a success",
			user:  &api.User{
				Username: "user1",
				Role:     api.UserGroupName,
			},
			leaseID: "abc123",
			expResp: response{
				StatusCode: 200,
				Body:       "{\"principalId\":\"user1\"}\n",
			},
			retLease: &lease.Lease{
				PrincipalID:              ptrString("user1"),
			},
			retErr:   nil,
		},
		{
			name:    "When user Get lease for other user service returns 401",
			user:  &api.User{
				Username: "user1",
				Role:     api.UserGroupName,
			},
			leaseID: "abc123",
			expResp: response{
				StatusCode: 401,
				Body:       "{\"error\":{\"code\":\"Unauthorized\",\"message\":\"Could not access the resource requested.\"}}",
			},
			retLease: &lease.Lease{
				PrincipalID:              ptrString("user2"),
			},
			retErr:   nil,
		},
		{
			name:    "When Get lease service returns a failure",
			user:  &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			leaseID: "abc123",
			expResp: response{
				StatusCode: 500,
				Body:       "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
			},
			retLease: nil,
			retErr:   fmt.Errorf("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", fmt.Sprintf("http://example.com/lease/%s", tt.leaseID), nil)

			r = mux.SetURLVars(r, map[string]string{
				"leaseID": tt.leaseID,
			})
			w := httptest.NewRecorder()

			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			leaseSvc := mocks.Servicer{}
			leaseSvc.On("Get", tt.leaseID).Return(
				tt.retLease, tt.retErr,
			)
			svcBldr.Config.WithService(&leaseSvc)
			_, err := svcBldr.Build()

			assert.Nil(t, err)
			if err == nil {
				Services = svcBldr
			}

			user = tt.user

			GetLeaseByID(w, r)

			resp := w.Result()
			body, err := ioutil.ReadAll(resp.Body)

			assert.Nil(t, err)
			assert.Equal(t, tt.expResp.StatusCode, resp.StatusCode)
			assert.Equal(t, tt.expResp.Body, string(body))
		})
	}
}

func TestHandler(t *testing.T){

	t.Run("When the handler invoking get and there are no errors", func(t *testing.T) {
		expectedLease := &lease.Lease{
			ID:             ptrString("unique-id"),
			AccountID:      ptrString("123456789"),
			PrincipalID:    ptrString("test"),
			Status:         lease.StatusActive.StatusPtr(),
			LastModifiedOn: ptrInt64(1561149393),
		}

		cfgBuilder := &config.ConfigurationBuilder{}
		svcBuilder := &config.ServiceBuilder{Config: cfgBuilder}

		leaseSvc := mocks.Servicer{}
		leaseSvc.On("Get", *expectedLease.ID).Return(
			expectedLease, nil,
		)

		userDetailerMock := apiMocks.UserDetailer{}
		userDetailerMock.On("GetUser", mock.Anything).Return(&api.User{
			Username: "",
			Role: api.AdminGroupName})

		svcBuilder.Config.WithService(&leaseSvc)
		svcBuilder.Config.WithService(&userDetailerMock)

		_, err := svcBuilder.Build()

		assert.Nil(t, err)
		if err == nil {
			Services = svcBuilder
		}

		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/leases/unique-id"}

		actualResponse, err := Handler(context.TODO(), mockRequest)
		assert.Nil(t, err)

		expectedResponse := MockAPIResponse(http.StatusOK, "{\"accountId\":\"123456789\",\"principalId\":\"test\",\"id\":\"unique-id\",\"leaseStatus\":\"Active\",\"lastModifiedOn\":1561149393}\n")
		assert.Equal(t, expectedResponse, actualResponse)
	})

	t.Run("When the handler invoking get and get fails", func(t *testing.T) {
		expectedLease := &lease.Lease{
			ID: ptrString("unique-id"),
		}

		expectedError := gErrors.New("error")
		cfgBuilder := &config.ConfigurationBuilder{}
		svcBuilder := &config.ServiceBuilder{Config: cfgBuilder}

		leaseSvc := mocks.Servicer{}
		leaseSvc.On("Get", *expectedLease.ID).Return(
			expectedLease, expectedError,
		)

		userDetailerMock := apiMocks.UserDetailer{}
		userDetailerMock.On("GetUser", mock.Anything).Return(&api.User{
			Username: "",
			Role: api.AdminGroupName})

		svcBuilder.Config.WithService(&leaseSvc)
		svcBuilder.Config.WithService(&userDetailerMock)
		_, err := svcBuilder.Build()

		assert.Nil(t, err)
		if err == nil {
			Services = svcBuilder
		}

		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/leases/unique-id"}

		actualResponse, err := Handler(context.TODO(), mockRequest)
		assert.Nil(t, err)

		expectedResponse := MockAPIErrorResponse(http.StatusInternalServerError, "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n")
		assert.Equal(t, expectedResponse, actualResponse)
	})
}

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func ptrInt64(i int64) *int64 {
	ptrI := i
	return &ptrI
}

func ptr64(i int64) *int64 {
	ptrI := i
	return &ptrI
}
