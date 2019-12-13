package api_test

import (
	"log"
	"net/url"
	"testing"

	"github.com/Optum/dce/pkg/api"
	"github.com/Optum/dce/pkg/errors"
	"github.com/Optum/dce/pkg/model"
	"github.com/gorilla/schema"

	"github.com/stretchr/testify/assert"
)

func ptrString(s string) *string {
	ptrS := s
	return &ptrS
}
func TestHelpers(t *testing.T) {
	readyStatus := model.Ready
	tests := []struct {
		name   string
		query  string
		i      interface{}
		result interface{}
		err    error
	}{
		{
			name:  "deserialize into account",
			query: "id=1",
			i:     &model.Account{},
			result: &model.Account{
				ID: ptrString("1"),
			},
			err: nil,
		},
		{
			name:   "deserialize into error",
			query:  "badField=1",
			i:      &model.Account{},
			result: &model.Account{},
			err: &errors.ErrValidation{
				Message: "error converting query parameters to struct",
				Err:     schema.MultiError{"badField": schema.UnknownKeyError{Key: "badField"}},
			},
		},
		{
			name:  "deserialize the accountStatus",
			query: "accountStatus=Ready",
			i:     &model.Account{},
			result: &model.Account{
				Status: &readyStatus,
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := url.ParseQuery(tt.query)
			if err != nil {
				log.Fatal(err)
			}
			err = api.GetStructFromQuery(tt.i, m)
			assert.Equal(t, tt.err, err)
			assert.Equal(t, tt.result, tt.i)
		})
	}
}
