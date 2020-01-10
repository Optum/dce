package errors

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIs(t *testing.T) {

	tests := []struct {
		name        string
		originalErr error
		expErr      error
		result      bool
	}{
		{
			name:        "is matches",
			originalErr: NewValidation("account", fmt.Errorf("wrapped error")),
			expErr:      NewValidation("account", fmt.Errorf("wrapped error")),
			result:      true,
		},
		{
			name:        "is doesn't match",
			originalErr: NewValidation("account", fmt.Errorf("wrapped error")),
			expErr:      NewInternalServer("failure", fmt.Errorf("wrapped error")),
			result:      false,
		},
		{
			name:        "is doesn't match on same error http codes",
			originalErr: NewInternalServer("fail", fmt.Errorf("wrapped error")),
			expErr:      NewInternalServer("failure", fmt.Errorf("wrapped error")),
			result:      false,
		},
		{
			name:        "is comparable",
			originalErr: fmt.Errorf("failure"),
			expErr:      fmt.Errorf("failure"),
			result:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			newErr := fmt.Errorf("new error: %w", tt.originalErr)
			assert.Equal(t, Is(newErr, tt.expErr), tt.result)
		})
	}

	t.Run("is returns false on nil", func(t *testing.T) {
		assert.False(t, Is(fmt.Errorf("new error"), nil))
	})

}

func TestAs(t *testing.T) {
	var errT errorT
	var errS StatusError

	tests := []struct {
		name   string
		err    error
		match  bool
		target interface{}
		want   error
	}{
		{
			name:   "as matches",
			err:    errorT{"T"},
			match:  true,
			target: &errT,
			want:   errorT{"T"},
		},
		{
			name:   "as matches with nested",
			err:    errorT{"T"},
			match:  true,
			target: &errT,
			want:   errorT{"T"},
		},
		{
			name:   "no match",
			err:    fmt.Errorf("failure"),
			match:  false,
			target: &errS,
			want:   StatusError{},
		},
	}

	for _, tt := range tests {
		rtarget := reflect.ValueOf(tt.target)
		rtarget.Elem().Set(reflect.Zero(reflect.TypeOf(tt.target).Elem()))
		t.Run(tt.name, func(t *testing.T) {
			fmt.Printf("%s\n", tt.name)
			match := As(tt.err, tt.target)

			assert.Equal(t, tt.match, match)
			got := rtarget.Elem().Interface()
			assert.Equal(t, tt.want, got)
		})
	}

}

type errorT struct{ s string }

func (e errorT) Error() string { return fmt.Sprintf("errorT(%s)", e.s) }

func (e errorT) Is(target error) bool {
	return e.Error() == target.Error()
}
