package api

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Optum/dce/pkg/dceerr"
	"github.com/stretchr/testify/assert"
)

func TestAPIWriting_Errors(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
		expectedJSON string
	}{
		{
			name:         "new validation error",
			err:          dceerr.NewValidation("resource", fmt.Errorf("wrapped error")),
			expectedCode: http.StatusBadRequest,
			expectedJSON: "{\"error\":{\"message\":\"resource validation error: wrapped error\",\"code\":\"RequestValidationError\"}}\n",
		},
		{
			name:         "new not found error",
			err:          dceerr.NewNotFound("resource", "name"),
			expectedCode: http.StatusNotFound,
			expectedJSON: "{\"error\":{\"message\":\"resource \\\"name\\\" not found\",\"code\":\"NotFoundError\"}}\n",
		},
		{
			name:         "new conflict error",
			err:          dceerr.NewConflict("resource", "name", fmt.Errorf("wrapped error")),
			expectedCode: http.StatusConflict,
			expectedJSON: "{\"error\":{\"message\":\"operation cannot be fulfilled on resource \\\"name\\\": wrapped error\",\"code\":\"ConflictError\"}}\n",
		},
		{
			name:         "new internal server error",
			err:          dceerr.NewInternalServer("failure message", fmt.Errorf("wrapped error")),
			expectedCode: http.StatusInternalServerError,
			expectedJSON: "{\"error\":{\"message\":\"failure message\",\"code\":\"ServerError\"}}\n",
		},
		{
			name:         "new unknown error",
			err:          errors.New("random error"),
			expectedCode: http.StatusInternalServerError,
			expectedJSON: "{\"error\":{\"message\":\"unknown error\",\"code\":\"ServerError\"}}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteAPIErrorResponse(w, tt.err)

			resp := w.Result()
			body, _ := ioutil.ReadAll(resp.Body)

			assert.Equal(t, tt.expectedCode, resp.StatusCode)
			assert.Equal(t, tt.expectedJSON, string(body))
		})
	}
}
