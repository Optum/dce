package main

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Optum/dce/pkg/account"
	"github.com/Optum/dce/pkg/account/accountiface/mocks"
	"github.com/Optum/dce/pkg/config"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gorilla/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetAccounts(t *testing.T) {

	type response struct {
		StatusCode int
		Body       string
	}
	tests := []struct {
		name             string
		expResp          response
		expLink          string
		query            *account.Account
		retAccounts      *account.Accounts
		retErr           error
		lastEvaluatedKey *account.LastEvaluatedKey
	}{
		{
			name:  "get all accounts",
			query: &account.Account{},
			expResp: response{
				StatusCode: 200,
				Body:       "[]\n",
			},
			retAccounts: &account.Accounts{},
			retErr:      nil,
		},
		{
			name:  "get paged accounts",
			query: &account.Account{},
			expResp: response{
				StatusCode: 200,
				Body:       "[{\"id\":\"123456789012\"}]\n",
			},
			retAccounts: &account.Accounts{
				account.Account{
					ID: ptrString("123456789012"),
				},
			},
			lastEvaluatedKey: &account.LastEvaluatedKey{
				ID: dynamodb.AttributeValue{
					S: ptrString("234567890123"),
				},
				AccountStatus: dynamodb.AttributeValue{
					S: ptrString("NotReady"),
				},
			},
			expLink: "<https://example.com/unit/accounts?limit=1&nextAccountStatus=NotReady&nextId=234567890123>; rel=\"next\"",
			retErr:  nil,
		},
		{
			name: "fail to get accounts",
			query: &account.Account{
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

			baseRequest = url.URL{}
			baseRequest.Scheme = "https"
			baseRequest.Host = "example.com"
			baseRequest.Path = fmt.Sprintf("%s%s", "unit", "/accounts")

			values := url.Values{}
			err := schema.NewEncoder().Encode(tt.query, values)

			assert.Nil(t, err)

			r.URL.RawQuery = values.Encode()
			w := httptest.NewRecorder()

			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			accountSvc := mocks.Servicer{}
			accountSvc.On("List", mock.MatchedBy(func(input *account.Account) bool {
				if (input.ID != nil && tt.query.ID != nil && *input.ID == *tt.query.ID) || input.ID == tt.query.ID {
					if tt.lastEvaluatedKey != nil {
						input.NextID = tt.lastEvaluatedKey.ID.S
						input.NextAccountStatus = tt.lastEvaluatedKey.AccountStatus.S

						input.Limit = ptr64(1)
					}
					return true
				}
				return false
			})).Return(
				tt.retAccounts, tt.retErr,
			)
			svcBldr.Config.WithService(&accountSvc)
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
			assert.Equal(t, tt.expLink, w.Header().Get("Link"))
		})
	}

}
