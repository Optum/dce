package dceerr

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errOriginal = errors.New("original error")
var errInternalServer = NewInternalServer("error", errOriginal)

func TestNew(t *testing.T) {

	tests := []struct {
		name               string
		err                *StatusError
		expectedMessage    string
		expectedCode       int
		expectedWrappedErr error
		expectedJSON       string
	}{
		{
			name:               "new validation error",
			err:                NewValidation("account", fmt.Errorf("wrapped error")),
			expectedMessage:    "account validation error: wrapped error",
			expectedCode:       http.StatusBadRequest,
			expectedWrappedErr: fmt.Errorf("wrapped error"),
			expectedJSON:       "{\"error\":{\"message\":\"account validation error: wrapped error\",\"code\":\"RequestValidationError\"}}\n",
		},
		{
			name:               "new not found error",
			err:                NewNotFound("resource", "name"),
			expectedMessage:    "resource \"name\" not found",
			expectedCode:       http.StatusNotFound,
			expectedWrappedErr: nil,
			expectedJSON:       "{\"error\":{\"message\":\"resource \\\"name\\\" not found\",\"code\":\"NotFoundError\"}}\n",
		},
		{
			name:               "new conflict error",
			err:                NewConflict("resource", "name", fmt.Errorf("wrapped error")),
			expectedMessage:    "operation cannot be fulfilled on resource \"name\": wrapped error",
			expectedCode:       http.StatusConflict,
			expectedWrappedErr: fmt.Errorf("wrapped error"),
			expectedJSON:       "{\"error\":{\"message\":\"operation cannot be fulfilled on resource \\\"name\\\": wrapped error\",\"code\":\"ConflictError\"}}\n",
		},
		{
			name:               "new internal server error",
			err:                NewInternalServer("failure message", fmt.Errorf("wrapped error")),
			expectedMessage:    "failure message",
			expectedCode:       http.StatusInternalServerError,
			expectedWrappedErr: fmt.Errorf("wrapped error"),
			expectedJSON:       "{\"error\":{\"message\":\"failure message\",\"code\":\"ServerError\"}}\n",
		},
		{
			name:               "new bad requst",
			err:                NewBadRequest("failure message"),
			expectedMessage:    "failure message",
			expectedCode:       http.StatusBadRequest,
			expectedWrappedErr: nil,
			expectedJSON:       "{\"error\":{\"message\":\"failure message\",\"code\":\"ClientError\"}}\n",
		},
		{
			name:               "new service unavailable",
			err:                NewServiceUnavailable("failure message"),
			expectedMessage:    "failure message",
			expectedCode:       http.StatusServiceUnavailable,
			expectedWrappedErr: nil,
			expectedJSON:       "{\"error\":{\"message\":\"failure message\",\"code\":\"ServerError\"}}\n",
		},
		{
			name:               "new already exists error",
			err:                NewAlreadyExists("account", "abc123"),
			expectedMessage:    "account \"abc123\" already exists",
			expectedCode:       http.StatusConflict,
			expectedWrappedErr: nil,
			expectedJSON:       "{\"error\":{\"message\":\"account \\\"abc123\\\" already exists\",\"code\":\"AlreadyExistsError\"}}\n",
		},
		{
			name:               "new admin role not assumable",
			err:                NewAdminRoleNotAssumable("roleArn", fmt.Errorf("wrapped error")),
			expectedMessage:    "adminRole \"roleArn\" is not assumable by the parent account",
			expectedCode:       http.StatusUnprocessableEntity,
			expectedWrappedErr: fmt.Errorf("wrapped error"),
			expectedJSON:       "{\"error\":{\"message\":\"adminRole \\\"roleArn\\\" is not assumable by the parent account\",\"code\":\"RequestValidationError\"}}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedMessage, tt.err.Error())
			assert.Equal(t, tt.expectedCode, APICodeForError(tt.err))
			assert.Equal(t, tt.err.OriginalError(), tt.expectedWrappedErr)

			var b bytes.Buffer
			json.NewEncoder(&b).Encode(tt.err)
			assert.Equal(
				t,
				tt.expectedJSON,
				b.String(),
			)
			assert.NotNil(
				t,
				GetStackTraceForError(tt.err),
			)
		})
	}
}

func TestFrameFormat(t *testing.T) {
	var tests = []struct {
		err    error
		format string
		want   string
	}{
		{
			errInternalServer,
			"%s",
			"error",
		},
		{
			errInternalServer,
			"%q",
			"\"error\"",
		},
		{
			errInternalServer,
			"%+v",
			"original error\n" +
				"github.com/Optum/dce/pkg/dceerr.init\n" +
				"\t.+/.*/error_test.go:17\n",
		},
	}

	for i, tt := range tests {
		testFormatRegexp(t, i, tt.err, tt.format, tt.want)
	}
}

func TestErrors_Cause(t *testing.T) {
	err1 := NewInternalServer("failure message", fmt.Errorf("original error"))
	err2 := fmt.Errorf("wrapped error1: %w", err1)
	err3 := fmt.Errorf("wrapped error2: %w", err2)

	assert.Equal(t, err1, Cause(err3))
}

func TestErrors_NotStatusErrors(t *testing.T) {
	err := errors.New("failure")

	assert.Equal(t, http.StatusInternalServerError, APICodeForError(err))
	assert.Nil(t, GetStackTraceForError(err))
}

func testFormatRegexp(t *testing.T, n int, arg interface{}, format, want string) {
	t.Helper()
	got := fmt.Sprintf(format, arg)
	gotLines := strings.SplitN(got, "\n", -1)
	wantLines := strings.SplitN(want, "\n", -1)

	if len(wantLines) > len(gotLines) {
		t.Errorf("test %d: wantLines(%d) > gotLines(%d):\n got: %q\nwant: %q", n+1, len(wantLines), len(gotLines), got, want)
		return
	}

	for i, w := range wantLines {
		match, err := regexp.MatchString(w, gotLines[i])
		if err != nil {
			t.Fatal(err)
		}
		if !match {
			t.Errorf("test %d: line %d: fmt.Sprintf(%q, err):\n got: %q\nwant: %q", n+1, i+1, format, got, want)
		}
	}
}
