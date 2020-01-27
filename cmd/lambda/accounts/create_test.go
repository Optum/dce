package main

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/account/accountiface/mocks"
	"github.com/Optum/dce/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreate(t *testing.T) {

	type response struct {
		StatusCode int
		Body       string
	}
	tests := []struct {
		name       string
		expResp    response
		reqBody    string
		retAccount *account.Account
		retErr     error
	}{
		{
			name: "success",
			expResp: response{
				StatusCode: 201,
				Body:       "{}\n",
			},
			reqBody:    "{ \"id\": \"123456789012\", \"adminRoleArn\": \"arn:test\" }",
			retAccount: &account.Account{},
			retErr:     nil,
		},
		{
			name: "failure on bad syntax",
			expResp: response{
				StatusCode: 400,
				Body:       "{\"error\":{\"message\":\"invalid request parameters\",\"code\":\"ClientError\"}}\n",
			},
			reqBody:    "{ \"id: \"123456789012\", \"adminRoleArn\": \"arn:test\" }",
			retAccount: &account.Account{},
			retErr:     nil,
		},
		{
			name:    "failure",
			reqBody: "{ \"id\": \"123456789012\", \"adminRoleArn\": \"arn:test\" }",
			expResp: response{
				StatusCode: 500,
				Body:       "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
			},
			retAccount: nil,
			retErr:     fmt.Errorf("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyReader := strings.NewReader(tt.reqBody)
			r := httptest.NewRequest("POST", "http://example.com/accounts", bodyReader)

			w := httptest.NewRecorder()

			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			accountSvc := mocks.Servicer{}
			accountSvc.On("Create", mock.AnythingOfType("*account.Account")).Return(
				tt.retAccount, tt.retErr,
			)
			svcBldr.Config.WithService(&accountSvc)
			_, err := svcBldr.Build()

			assert.Nil(t, err)
			if err == nil {
				Services = svcBldr
			}

			CreateAccount(w, r)

			resp := w.Result()
			body, err := ioutil.ReadAll(resp.Body)

			assert.Nil(t, err)
			assert.Equal(t, tt.expResp.StatusCode, resp.StatusCode)
			assert.Equal(t, tt.expResp.Body, string(body))
		})
	}

}
