package main

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	amMocks "github.com/Optum/dce/pkg/accountmanager/accountmanageriface/mocks"
	"github.com/Optum/dce/pkg/config"
	dataMocks "github.com/Optum/dce/pkg/data/dataiface/mocks"
	"github.com/Optum/dce/pkg/model"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		name          string
		expResp       response
		reqBody       string
		accountID     string
		getRetAccount *model.Account
		getRetErr     error
		writeRetErr   error
	}{
		{
			name:      "success",
			accountID: "123456789012",
			reqBody:   fmt.Sprintf("{\"metadata\": {\"key\": \"value\"}}"),
			expResp: response{
				StatusCode: 200,
				Body: fmt.Sprintf("{\"id\":\"123456789012\",\"accountStatus\":\"Ready\",\"lastModifiedOn\":%d,\"createdOn\":%d,\"adminRoleArn\":\"arn:aws:iam::123456789012:role/test\",\"metadata\":{\"key\":\"value\"}}\n",
					now, now),
			},
			getRetAccount: &model.Account{
				ID:             ptrString("123456789012"),
				Status:         model.AccountStatusReady.AccountStatusPtr(),
				AdminRoleArn:   ptrString("arn:aws:iam::123456789012:role/test"),
				CreatedOn:      &now,
				LastModifiedOn: &now,
			},
			getRetErr: nil,
		},
		{
			name:      "failure db",
			accountID: "123456789012",
			reqBody:   fmt.Sprintf("{\"metadata\": {\"key\": \"value\"}}"),
			expResp: response{
				StatusCode: 500,
				Body:       "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
			},
			getRetAccount: nil,
			getRetErr:     fmt.Errorf("failure"),
		},
		{
			name:      "failure decode",
			accountID: "123456789012",
			reqBody:   fmt.Sprintf("{\"metadata\": \"key\": \"value\"}}"),
			expResp: response{
				StatusCode: 400,
				Body:       "{\"error\":{\"message\":\"invalid request parameters\",\"code\":\"ClientError\"}}\n",
			},
			getRetAccount: nil,
			getRetErr:     fmt.Errorf("failure"),
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

			dataSvc := dataMocks.AccountData{}
			dataSvc.On("GetAccountByID", tt.accountID).Return(
				tt.getRetAccount, tt.getRetErr,
			)
			dataSvc.On("WriteAccount", mock.AnythingOfType("*model.Account"), &now).Return(
				tt.writeRetErr,
			)

			amSvc := amMocks.AccountManagerAPI{}

			svcBldr.Config.WithService(&dataSvc).WithService(&amSvc)
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
