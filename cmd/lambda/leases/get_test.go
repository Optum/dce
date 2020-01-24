package main

import (
	"testing"

	"fmt"
	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/lease/leaseiface/mocks"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http/httptest"
)

func TestGetLeaseByID(t *testing.T) {

	type response struct {
		StatusCode int
		Body       string
	}
	tests := []struct {
		name     string
		expResp  response
		leaseID  string
		retLease *lease.Lease
		retErr   error
	}{
		{
			name:    "success",
			leaseID: "abc123",
			expResp: response{
				StatusCode: 200,
				Body:       "{}\n",
			},
			retLease: &lease.Lease{},
			retErr:   nil,
		},
		{
			name:    "failure",
			leaseID: "abc123",
			expResp: response{
				StatusCode: 500,
				Body:       "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
			},
			retLease: nil,
			retErr:   fmt.Errorf("failure"),
		},
		{
			name:    "found more than one",
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

			GetLeaseByID(w, r)

			resp := w.Result()
			body, err := ioutil.ReadAll(resp.Body)

			assert.Nil(t, err)
			assert.Equal(t, tt.expResp.StatusCode, resp.StatusCode)
			assert.Equal(t, tt.expResp.Body, string(body))
		})
	}

}
