package main

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Optum/dce/pkg/config"
	"github.com/Optum/dce/pkg/lease"
	"github.com/Optum/dce/pkg/lease/leaseiface/mocks"
	"github.com/gorilla/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetLeases(t *testing.T) {

	type response struct {
		StatusCode int
		Body       string
	}
	tests := []struct {
		name            string
		expResp         response
		expLink         string
		query           *lease.Lease
		retLeases       *lease.Leases
		retErr          error
		nextAccountID   *string
		nextPrincipalID *string
	}{
		{
			name:  "get all leases",
			query: &lease.Lease{},
			expResp: response{
				StatusCode: 200,
				Body:       "[]\n",
			},
			retLeases: &lease.Leases{},
			retErr:    nil,
		},
		{
			name:  "get paged leases",
			query: &lease.Lease{},
			expResp: response{
				StatusCode: 200,
				Body:       "[{\"accountId\":\"123456789012\",\"principalId\":\"User1\"}]\n",
			},
			retLeases: &lease.Leases{
				lease.Lease{
					AccountID:   ptrString("123456789012"),
					PrincipalID: ptrString("User1"),
				},
			},
			nextAccountID:   ptrString("234567890123"),
			nextPrincipalID: ptrString("User2"),
			expLink:         "<https://example.com/unit/leases?limit=1&nextAccountId=234567890123&nextPrincipalId=User2>; rel=\"next\"",
			retErr:          nil,
		},
		{
			name: "fail to get leases",
			query: &lease.Lease{
				AccountID:   ptrString("abc123"),
				PrincipalID: ptrString("User2"),
			},
			expResp: response{
				StatusCode: 500,
				Body:       "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
			},
			retLeases: nil,
			retErr:    fmt.Errorf("failure"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://example.com/leases", nil)

			baseRequest = url.URL{}
			baseRequest.Scheme = "https"
			baseRequest.Host = "example.com"
			baseRequest.Path = fmt.Sprintf("%s%s", "unit", "/leases")

			values := url.Values{}
			err := schema.NewEncoder().Encode(tt.query, values)
			assert.Nil(t, err)

			r.URL.RawQuery = values.Encode()
			w := httptest.NewRecorder()

			cfgBldr := &config.ConfigurationBuilder{}
			svcBldr := &config.ServiceBuilder{Config: cfgBldr}

			leaseSvc := mocks.Servicer{}

			leaseSvc.On("List", mock.MatchedBy(func(input *lease.Lease) bool {
				if (input.AccountID != nil && tt.query.AccountID != nil && *input.AccountID == *tt.query.AccountID) || input.AccountID == tt.query.AccountID {
					if tt.nextAccountID != nil && tt.nextPrincipalID != nil {
						input.NextAccountID = tt.nextAccountID
						input.NextPrincipalID = tt.nextPrincipalID
						input.Limit = ptr64(1)
					}
					return true
				}
				return false
			})).Return(
				tt.retLeases, tt.retErr,
			)
			svcBldr.Config.WithService(&leaseSvc)
			_, err = svcBldr.Build()

			assert.Nil(t, err)
			if err == nil {
				Services = svcBldr
			}

			GetLeases(w, r)

			resp := w.Result()
			body, err := ioutil.ReadAll(resp.Body)

			assert.Nil(t, err)
			assert.Equal(t, tt.expResp.StatusCode, resp.StatusCode)
			assert.Equal(t, tt.expResp.Body, string(body))
			assert.Equal(t, tt.expLink, w.Header().Get("Link"))
		})
	}

}
