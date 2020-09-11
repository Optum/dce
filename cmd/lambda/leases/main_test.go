package main

import (
	"github.com/Optum/dce/pkg/api"
	apiMocks "github.com/Optum/dce/pkg/api/mocks"
	"github.com/Optum/dce/pkg/lease"
	"github.com/stretchr/testify/mock"

	"os"
	"testing"

	"context"
	gErrors "errors"
	"net/http"

	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/lease/leaseiface/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// MockAPIResponse is a helper function to create and return a valid response
// for an API Gateway
func MockAPIResponse(status int, body string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		MultiValueHeaders: map[string][]string{
			"Content-Type":                []string{"application/json"},
			"Access-Control-Allow-Origin": []string{"*"},
		},
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
			"Content-Type":                "application/json",
		},
		Body: body,
	}
}

func MockAPIErrorResponse(status int, body string) events.APIGatewayProxyResponse {

	return events.APIGatewayProxyResponse{
		StatusCode: status,
		MultiValueHeaders: map[string][]string{
			"Content-Type":                []string{"application/json"},
			"Access-Control-Allow-Origin": []string{"*"},
		},
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
			"Content-Type":                "application/json",
		},
		Body: body,
	}
}

func TestList(t *testing.T) {
	t.Run("When the handler invoking lease controller - list by status and there are no errors", func(t *testing.T) {
		lease1 := &lease.Lease{
			ID:             ptrString("unique-id"),
			AccountID:      ptrString("123456789"),
			PrincipalID:    ptrString("test"),
			Status:         lease.StatusActive.StatusPtr(),
			LastModifiedOn: ptrInt64(1561149393),
		}

		expectedLeases := &lease.Leases{*lease1}

		cfgBuilder := &config.ConfigurationBuilder{}
		svcBuilder := &config.ServiceBuilder{Config: cfgBuilder}

		leaseSvc := mocks.Servicer{}
		leaseSvc.On("List", mock.Anything).Return(
			expectedLeases, nil,
		)
		userDetailersvc := apiMocks.UserDetailer{}
		userDetailersvc.On("GetUser", mock.Anything).Return(&api.User{
			Username: "",
			Role:     api.AdminGroupName})

		svcBuilder.Config.WithService(&leaseSvc)
		svcBuilder.Config.WithService(&userDetailersvc)
		_, err := svcBuilder.Build()

		assert.Nil(t, err)
		if err == nil {
			Services = svcBuilder
		}

		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/leases?status=active"}

		actualResponse, err := Handler(context.TODO(), mockRequest)
		assert.Nil(t, err)

		expectedResponse := MockAPIResponse(http.StatusOK, "[{\"accountId\":\"123456789\",\"principalId\":\"test\",\"id\":\"unique-id\",\"leaseStatus\":\"Active\",\"lastModifiedOn\":1561149393}]\n")
		assert.Equal(t, expectedResponse, actualResponse)
	})

	t.Run("When the handler invoking lease controller - list and there are no errors", func(t *testing.T) {
		lease1 := &lease.Lease{
			ID:             ptrString("unique-id"),
			AccountID:      ptrString("123456789"),
			PrincipalID:    ptrString("test"),
			Status:         lease.StatusActive.StatusPtr(),
			LastModifiedOn: ptrInt64(1561149393),
		}

		expectedLeases := &lease.Leases{*lease1}

		cfgBuilder := &config.ConfigurationBuilder{}
		svcBuilder := &config.ServiceBuilder{Config: cfgBuilder}

		leaseSvc := mocks.Servicer{}
		leaseSvc.On("List", mock.Anything).Return(
			expectedLeases, nil,
		)
		userDetailersvc := apiMocks.UserDetailer{}
		userDetailersvc.On("GetUser", mock.Anything).Return(&api.User{
			Username: "",
			Role:     api.AdminGroupName})

		svcBuilder.Config.WithService(&leaseSvc)
		svcBuilder.Config.WithService(&userDetailersvc)
		_, err := svcBuilder.Build()

		assert.Nil(t, err)
		if err == nil {
			Services = svcBuilder
		}

		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/leases"}

		actualResponse, err := Handler(context.TODO(), mockRequest)
		assert.Nil(t, err)

		expectedResponse := MockAPIResponse(http.StatusOK, "[{\"accountId\":\"123456789\",\"principalId\":\"test\",\"id\":\"unique-id\",\"leaseStatus\":\"Active\",\"lastModifiedOn\":1561149393}]\n")
		assert.Equal(t, expectedResponse, actualResponse)
	})

	t.Run("When the handler invoking lease controller - list and get fails", func(t *testing.T) {
		expectedError := gErrors.New("error")
		cfgBuilder := &config.ConfigurationBuilder{}
		svcBuilder := &config.ServiceBuilder{Config: cfgBuilder}

		leaseSvc := mocks.Servicer{}
		leaseSvc.On("List", mock.Anything).Return(
			nil, expectedError,
		)
		userDetailersvc := apiMocks.UserDetailer{}
		userDetailersvc.On("GetUser", mock.Anything).Return(&api.User{
			Username: "",
			Role:     api.AdminGroupName})

		svcBuilder.Config.WithService(&leaseSvc)
		svcBuilder.Config.WithService(&userDetailersvc)
		_, err := svcBuilder.Build()

		assert.Nil(t, err)
		if err == nil {
			Services = svcBuilder
		}

		mockRequest := events.APIGatewayProxyRequest{HTTPMethod: http.MethodGet, Path: "/leases"}

		actualResponse, err := Handler(context.TODO(), mockRequest)
		assert.Nil(t, err)

		expectedResponse := MockAPIErrorResponse(http.StatusInternalServerError, "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n")
		assert.Equal(t, expectedResponse, actualResponse)
	})
}
