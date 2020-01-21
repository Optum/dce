package main

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/account/accountiface/mocks"
	"github.com/Optum/dce/pkg/config"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func TestUpdateAccountByID(t *testing.T) {

	now := time.Now().Unix()
	type response struct {
		StatusCode int
		Body       string
	}
	tests := []struct {
		name        string
		expResp     response
		reqBody     string
		reqAccount  *account.Account
		accountID   string
		retAccount  *account.Account
		retErr      error
		writeRetErr error
	}{
		{
			name:      "success",
			accountID: "123456789012",
			reqBody:   fmt.Sprintf("{\"metadata\": {\"key\": \"value\"}}"),
			reqAccount: &account.Account{
				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
			expResp: response{
				StatusCode: 200,
				Body: fmt.Sprintf("{\"id\":\"123456789012\",\"accountStatus\":\"Ready\",\"lastModifiedOn\":%d,\"createdOn\":%d,\"adminRoleArn\":\"arn:aws:iam::123456789012:role/test\",\"metadata\":{\"key\":\"value\"}}\n",
					now, now),
			},
			retAccount: &account.Account{
				ID:           ptrString("123456789012"),
				Status:       account.StatusReady.StatusPtr(),
				AdminRoleArn: ptrString("arn:aws:iam::123456789012:role/test"),
				Metadata: map[string]interface{}{
					"key": "value",
				},
				CreatedOn:      &now,
				LastModifiedOn: &now,
			},
			retErr: nil,
		},
		{
			name:      "failure db",
			accountID: "123456789012",
			reqBody:   fmt.Sprintf("{\"metadata\": {\"key\": \"value\"}}"),
			reqAccount: &account.Account{
				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
			expResp: response{
				StatusCode: 500,
				Body:       "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
			},
			retAccount: nil,
			retErr:     fmt.Errorf("failure"),
		},
		{
			name:      "failure decode",
			accountID: "123456789012",
			reqBody:   fmt.Sprintf("{\"metadata\": \"key\": \"value\"}}"),
			reqAccount: &account.Account{
				Metadata: map[string]interface{}{
					"key": "value",
				},
			},
			expResp: response{
				StatusCode: 400,
				Body:       "{\"error\":{\"message\":\"invalid request parameters\",\"code\":\"ClientError\"}}\n",
			},
			retAccount: nil,
			retErr:     fmt.Errorf("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			r := httptest.NewRequest(
				"POST",
				fmt.Sprintf("http://example.com/accounts/%s", tt.accountID),
				strings.NewReader(fmt.Sprintf(tt.reqBody)),
			)

			r = mux.SetURLVars(r, map[string]string{
				"accountId": tt.accountID,
			})
			w := httptest.NewRecorder()

			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			accountSvc := mocks.Servicer{}
			accountSvc.On("Update", tt.accountID, tt.reqAccount).Return(
				tt.retAccount, tt.retErr,
			)

			svcBldr.Config.WithService(&accountSvc)
			_, err := svcBldr.Build()

			assert.Nil(t, err)
			if err == nil {
				Services = svcBldr
			}

			UpdateAccountByID(w, r)

			resp := w.Result()
			body, err := ioutil.ReadAll(resp.Body)

			assert.Nil(t, err)
			assert.Equal(t, tt.expResp.StatusCode, resp.StatusCode)
			assert.JSONEq(t, tt.expResp.Body, string(body))
		})
	}

}
