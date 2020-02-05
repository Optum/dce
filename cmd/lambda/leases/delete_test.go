package main

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/lease/leaseiface/mocks"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestDeleteLease(t *testing.T) {

	type response struct {
		StatusCode int
		Body       string
	}
	tests := []struct {
		name          string
		expResp       response
		leaseID       string
		getErr        error
		expLease      *lease.Lease
		transitionErr error
	}{
		{
			name:    "successful delete",
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
			name:    "When Delete lease service returns a failure",
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
			r := httptest.NewRequest("DELETE", fmt.Sprintf("http://example.com/lease/%s", tt.leaseID), nil)

			r = mux.SetURLVars(r, map[string]string{
				"leaseID": tt.leaseID,
			})
			w := httptest.NewRecorder()
			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			leaseSvc := mocks.Servicer{}
			leaseSvc.On("Delete", tt.leaseID).Return(
				tt.expLease, tt.getErr,
			)

			svcBldr.Config.WithService(&leaseSvc)
			_, err := svcBldr.Build()

			assert.Nil(t, err)
			if err == nil {
				Services = svcBldr
			}

			DeleteLeaseByID(w, r)

			resp := w.Result()
			body, err := ioutil.ReadAll(resp.Body)

			assert.Nil(t, err)
			assert.Equal(t, tt.expResp.StatusCode, resp.StatusCode)
			assert.Equal(t, tt.expResp.Body, string(body))
		})
	}

}
