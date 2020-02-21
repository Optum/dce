package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Optum/dce/pkg/api"
	apiMocks "github.com/Optum/dce/pkg/api/mocks"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/lease/leaseiface/mocks"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"
)

func TestDeleteLeaseByLeaseID(t *testing.T) {

	type response struct {
		StatusCode int
		Body       string
	}
	tests := []struct {
		name          string
		user          *api.User
		expResp       response
		leaseID       string
		getErr        error
		expLease      *lease.Lease
		transitionErr error
	}{
		{
			name: "admin successfully deletes other users lease",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			leaseID: "abc123",
			expResp: response{
				StatusCode: 200,
				Body:       "{\"accountId\":\"123456789012\",\"principalId\":\"principal\",\"id\":\"abc123\",\"leaseStatus\":\"Inactive\",\"leaseStatusReason\":\"Expired\"}\n",
			},
			expLease: &lease.Lease{
				ID:           ptrString("abc123"),
				Status:       lease.StatusInactive.StatusPtr(),
				StatusReason: lease.StatusReasonExpired.StatusReasonPtr(),
				PrincipalID:  ptrString("principal"),
				AccountID:    ptrString("123456789012"),
			},
			getErr: nil,
		},
		{
			name: "user successfully deletes their own lease",
			user: &api.User{
				Username: "user1",
				Role:     api.UserGroupName,
			},
			leaseID: "abc123",
			expResp: response{
				StatusCode: 200,
				Body:       "{\"accountId\":\"123456789012\",\"principalId\":\"user1\",\"id\":\"abc123\",\"leaseStatus\":\"Inactive\",\"leaseStatusReason\":\"Expired\"}\n",
			},
			expLease: &lease.Lease{
				ID:           ptrString("abc123"),
				Status:       lease.StatusInactive.StatusPtr(),
				StatusReason: lease.StatusReasonExpired.StatusReasonPtr(),
				PrincipalID:  ptrString("user1"),
				AccountID:    ptrString("123456789012"),
			},
			getErr: nil,
		},
		{
			name: "user cannot delete other users lease",
			user: &api.User{
				Username: "user1",
				Role:     api.UserGroupName,
			},
			leaseID: "abc123",
			expResp: response{
				StatusCode: 401,
				Body:       "{\"error\":{\"message\":\"User [user1] with role: [User] attempted to act on a lease for [user2], but was not authorized\",\"code\":\"UnauthorizedError\"}}\n",
			},
			expLease: &lease.Lease{
				ID:           ptrString("abc123"),
				Status:       lease.StatusInactive.StatusPtr(),
				StatusReason: lease.StatusReasonExpired.StatusReasonPtr(),
				PrincipalID:  ptrString("user2"),
				AccountID:    ptrString("123456789012"),
			},
			getErr: nil,
		},
		{
			name: "When Admin Delete lease service returns a failure",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			leaseID: "abc123",
			expResp: response{
				StatusCode: 500,
				Body:       "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
			},
			expLease: &lease.Lease{
				ID:           ptrString("abc123"),
				Status:       lease.StatusInactive.StatusPtr(),
				StatusReason: lease.StatusReasonExpired.StatusReasonPtr(),
				PrincipalID:  ptrString("principal"),
				AccountID:    ptrString("123456789012"),
			},
			getErr: fmt.Errorf("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			leaseSvc := mocks.Servicer{}
			leaseSvc.On("Get", tt.leaseID).Return(
				tt.expLease, tt.getErr,
			)
			leaseSvc.On("Delete", tt.leaseID).Return(
				tt.expLease, tt.getErr,
			)

			userDetailSvc := apiMocks.UserDetailer{}
			userDetailSvc.On("GetUser", mock.Anything).Return(tt.user)

			svcBldr.Config.WithService(&userDetailSvc)
			svcBldr.Config.WithService(&leaseSvc)
			_, err := svcBldr.Build()

			assert.Nil(t, err)
			if err == nil {
				Services = svcBldr
			}

			mockRequest := events.APIGatewayProxyRequest{
				Path:           "/leases/" + tt.leaseID,
				HTTPMethod:     http.MethodDelete,
				RequestContext: events.APIGatewayProxyRequestContext{},
			}
			actualResponse, err := Handler(context.TODO(), mockRequest)

			assert.Nil(t, err)
			assert.Equal(t, tt.expResp.StatusCode, actualResponse.StatusCode)
			assert.Equal(t, tt.expResp.Body, actualResponse.Body)
		})
	}

}

func TestDeleteLeaseByPrincipalIDAndAccountID(t *testing.T) {

	type response struct {
		StatusCode int
		Body       string
	}
	tests := []struct {
		name       string
		user       *api.User
		inputLease *lease.Lease
		getLeases  *lease.Leases
		expResp    response
		getErr     error
		expLease   *lease.Lease
	}{
		{
			name: "admin successfully deletes other users lease",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			inputLease: &lease.Lease{
				PrincipalID: ptrString("User1"),
				AccountID:   ptrString("123456789012"),
			},
			getLeases: &lease.Leases{
				lease.Lease{
					ID:          ptrString("abc123"),
					AccountID:   ptrString("123456789012"),
					PrincipalID: ptrString("User1"),
				},
			},
			expResp: response{
				StatusCode: 200,
				Body:       "{\"accountId\":\"123456789012\",\"principalId\":\"User1\",\"id\":\"abc123\",\"leaseStatus\":\"Inactive\",\"leaseStatusReason\":\"Expired\"}\n",
			},
			expLease: &lease.Lease{
				ID:           ptrString("abc123"),
				Status:       lease.StatusInactive.StatusPtr(),
				StatusReason: lease.StatusReasonExpired.StatusReasonPtr(),
				PrincipalID:  ptrString("User1"),
				AccountID:    ptrString("123456789012"),
			},
			getErr: nil,
		},
		{
			name: "user successful deletes their own  lease",
			user: &api.User{
				Username: "user1",
				Role:     api.UserGroupName,
			},
			inputLease: &lease.Lease{
				PrincipalID: ptrString("user1"),
				AccountID:   ptrString("123456789012"),
			},
			getLeases: &lease.Leases{
				lease.Lease{
					ID:          ptrString("abc123"),
					AccountID:   ptrString("123456789012"),
					PrincipalID: ptrString("user1"),
				},
			},
			expResp: response{
				StatusCode: 200,
				Body:       "{\"accountId\":\"123456789012\",\"principalId\":\"user1\",\"id\":\"abc123\",\"leaseStatus\":\"Inactive\",\"leaseStatusReason\":\"Expired\"}\n",
			},
			expLease: &lease.Lease{
				ID:           ptrString("abc123"),
				Status:       lease.StatusInactive.StatusPtr(),
				StatusReason: lease.StatusReasonExpired.StatusReasonPtr(),
				PrincipalID:  ptrString("user1"),
				AccountID:    ptrString("123456789012"),
			},
			getErr: nil,
		},
		{
			name: "user cannot delete another users lease",
			user: &api.User{
				Username: "user1",
				Role:     api.UserGroupName,
			},
			inputLease: &lease.Lease{
				PrincipalID: ptrString("user2"),
				AccountID:   ptrString("123456789012"),
			},
			getLeases: &lease.Leases{
				lease.Lease{
					ID:          ptrString("123"),
					AccountID:   ptrString("123456789012"),
					PrincipalID: ptrString("user2"),
				},
			},
			expResp: response{
				StatusCode: 401,
				Body:       "{\"error\":{\"message\":\"User [user1] with role: [User] attempted to act on a lease for [user2], but was not authorized\",\"code\":\"UnauthorizedError\"}}\n",
			},
			expLease: &lease.Lease{
				ID:           ptrString("abc123"),
				Status:       lease.StatusInactive.StatusPtr(),
				StatusReason: lease.StatusReasonExpired.StatusReasonPtr(),
				PrincipalID:  ptrString("user2"),
				AccountID:    ptrString("123456789012"),
			},
			getErr: nil,
		},
		{
			name: "when delete input is missing accountID",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			inputLease: &lease.Lease{
				PrincipalID: ptrString("principal"),
			},
			getLeases: &lease.Leases{
				lease.Lease{
					ID:          ptrString("123"),
					AccountID:   ptrString("123456789012"),
					PrincipalID: ptrString("User1"),
				},
			},
			expResp: response{
				StatusCode: 400,
				Body:       "{\"error\":{\"message\":\"invalid request parameters: missing AccountID\",\"code\":\"ClientError\"}}\n",
			},
			expLease: &lease.Lease{
				ID:           ptrString("abc123"),
				Status:       lease.StatusInactive.StatusPtr(),
				StatusReason: lease.StatusReasonExpired.StatusReasonPtr(),
				PrincipalID:  ptrString("principal"),
				AccountID:    ptrString("123456789012"),
			},
			getErr: nil,
		},
		{
			name: "when delete input is missing principalID",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			inputLease: &lease.Lease{
				AccountID: ptrString("User1"),
			},
			getLeases: nil,
			expResp: response{
				StatusCode: 400,
				Body:       "{\"error\":{\"message\":\"invalid request parameters: missing PrincipalID\",\"code\":\"ClientError\"}}\n",
			},
			expLease: &lease.Lease{
				ID:           ptrString("abc123"),
				Status:       lease.StatusInactive.StatusPtr(),
				StatusReason: lease.StatusReasonExpired.StatusReasonPtr(),
				PrincipalID:  ptrString("principal"),
				AccountID:    ptrString("123456789012"),
			},
			getErr: nil,
		},
		{
			name: "when no matching lease found error",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			inputLease: &lease.Lease{
				PrincipalID: ptrString("principal"),
				AccountID:   ptrString("123456789012"),
			},
			getLeases: &lease.Leases{},
			expResp: response{
				StatusCode: 404,
				Body:       "{\"error\":{\"message\":\"lease \\\"with Principal ID principal and Account ID 123456789012\\\" not found\",\"code\":\"NotFoundError\"}}\n",
			},
			expLease: &lease.Lease{
				ID:           ptrString("abc123"),
				Status:       lease.StatusInactive.StatusPtr(),
				StatusReason: lease.StatusReasonExpired.StatusReasonPtr(),
				PrincipalID:  ptrString("principal"),
				AccountID:    ptrString("123456789012"),
			},
			getErr: nil,
		},
		{
			name: "when Delete lease service returns a failure",
			user: &api.User{
				Username: "admin1",
				Role:     api.AdminGroupName,
			},
			inputLease: &lease.Lease{
				PrincipalID: ptrString("principal"),
				AccountID:   ptrString("123456789012"),
			},
			expResp: response{
				StatusCode: 500,
				Body:       "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
			},
			expLease: &lease.Lease{
				ID:           ptrString("abc123"),
				Status:       lease.StatusInactive.StatusPtr(),
				StatusReason: lease.StatusReasonExpired.StatusReasonPtr(),
				PrincipalID:  ptrString("principal"),
				AccountID:    ptrString("123456789012"),
			},
			getErr: fmt.Errorf("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			leaseSvc := mocks.Servicer{}
			leaseSvc.On("List", mock.AnythingOfType("*lease.Lease")).Return(
				tt.getLeases, tt.getErr,
			)

			leaseSvc.On("Delete", *tt.expLease.ID).Return(
				tt.expLease, tt.getErr,
			)
			userDetailSvc := apiMocks.UserDetailer{}
			userDetailSvc.On("GetUser", mock.Anything).Return(tt.user)
			svcBldr.Config.WithService(&userDetailSvc)
			svcBldr.Config.WithService(&leaseSvc)
			_, err := svcBldr.Build()

			assert.Nil(t, err)
			if err == nil {
				Services = svcBldr
			}

			b := new(bytes.Buffer)
			err = json.NewEncoder(b).Encode(tt.inputLease)
			assert.Nil(t, err)
			mockRequest := events.APIGatewayProxyRequest{
				Path:           "/leases",
				HTTPMethod:     http.MethodDelete,
				RequestContext: events.APIGatewayProxyRequestContext{},
				Body:           b.String(),
			}
			actualResponse, err := Handler(context.TODO(), mockRequest)

			assert.Nil(t, err)
			assert.Equal(t, tt.expResp.StatusCode, actualResponse.StatusCode)
			assert.Equal(t, tt.expResp.Body, actualResponse.Body)
		})
	}

}
