package main

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/data/dataiface/mocks"
	"github.com/Optum/dce/pkg/model"
	"github.com/gorilla/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}

func TestGetAccounts(t *testing.T) {

	type response struct {
		StatusCode int
		Body       string
	}
	tests := []struct {
		name        string
		expResp     response
		query       *model.Account
		retAccounts *model.Accounts
		retErr      error
		nextID      *string
	}{
		{
			name:  "get all accounts",
			query: &model.Account{},
			expResp: response{
				StatusCode: 200,
				Body:       "[]\n",
			},
			retAccounts: &model.Accounts{},
			retErr:      nil,
		},
		{
			name: "fail to get accounts",
			query: &model.Account{
				ID: ptrString("abc123"),
			},
			expResp: response{
				StatusCode: 500,
				Body:       "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
			},
			retAccounts: nil,
			retErr:      fmt.Errorf("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://example.com/accounts", nil)

			values := url.Values{}
			err := schema.NewEncoder().Encode(tt.query, values)
			assert.Nil(t, err)

			r.URL.RawQuery = values.Encode()
			w := httptest.NewRecorder()

			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			dataSvc := mocks.AccountData{}
			dataSvc.On("GetAccounts", mock.MatchedBy(func(input *model.Account) bool {
				fmt.Printf("%+v, %+v, %+v\n", input.ID, tt.query.ID, input.ID == tt.query.ID)
				if (input.ID != nil && tt.query.ID != nil && *input.ID == *tt.query.ID) || input.ID == tt.query.ID {
					if tt.nextID != nil {
						input.NextID = tt.nextID
					}
					return true
				}
				return false
			})).Return(
				tt.retAccounts, tt.retErr,
			)
			svcBldr.Config.WithService(&dataSvc)
			_, err = svcBldr.Build()

			assert.Nil(t, err)
			if err == nil {
				Services = svcBldr
			}

			GetAccounts(w, r)

			resp := w.Result()
			body, err := ioutil.ReadAll(resp.Body)

			assert.Nil(t, err)
			assert.Equal(t, tt.expResp.StatusCode, resp.StatusCode)
			assert.Equal(t, tt.expResp.Body, string(body))
		})
	}

}
