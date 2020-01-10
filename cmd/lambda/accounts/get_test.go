package main

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/data/dataiface/mocks"
	"github.com/Optum/dce/pkg/model"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestGetAccountByID(t *testing.T) {

	type response struct {
		StatusCode int
		Body       string
	}
	tests := []struct {
		name       string
		expResp    response
		accountID  string
		retAccount *model.Account
		retErr     error
	}{
		{
			name:      "success",
			accountID: "abc123",
			expResp: response{
				StatusCode: 200,
				Body:       "{}\n",
			},
			retAccount: &model.Account{},
			retErr:     nil,
		},
		{
			name:      "failure",
			accountID: "abc123",
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
			r := httptest.NewRequest("GET", fmt.Sprintf("http://example.com/accounts/%s", tt.accountID), nil)

			r = mux.SetURLVars(r, map[string]string{
				"accountId": tt.accountID,
			})
			w := httptest.NewRecorder()

			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			dataSvc := mocks.AccountData{}
			dataSvc.On("GetAccountByID", tt.accountID).Return(
				tt.retAccount, tt.retErr,
			)
			svcBldr.Config.WithService(&dataSvc)
			_, err := svcBldr.Build()

			assert.Nil(t, err)
			if err == nil {
				Services = svcBldr
			}

			GetAccountByID(w, r)

			resp := w.Result()
			body, err := ioutil.ReadAll(resp.Body)

			assert.Nil(t, err)
			assert.Equal(t, tt.expResp.StatusCode, resp.StatusCode)
			assert.Equal(t, tt.expResp.Body, string(body))
		})
	}

}
