package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIs(t *testing.T) {

	tests := []struct {
		name        string
		originalErr error
		isErr       error
		result      bool
	}{
		{
			name:        "is matches",
			originalErr: NewValidation("account", fmt.Errorf("wrapped error")),
			isErr:       NewValidation("account", fmt.Errorf("wrapped error")),
			result:      true,
		},
		{
			name:        "is doesn't match",
			originalErr: NewValidation("account", fmt.Errorf("wrapped error")),
			isErr:       NewInternalServer("failure", fmt.Errorf("wrapped error")),
			result:      false,
		},
		{
			name:        "is doesn't match on same error http codes",
			originalErr: NewInternalServer("fail", fmt.Errorf("wrapped error")),
			isErr:       NewInternalServer("failure", fmt.Errorf("wrapped error")),
			result:      false,
		},
		{
			name:        "is comparable",
			originalErr: fmt.Errorf("failure"),
			isErr:       fmt.Errorf("failure"),
			result:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			newErr := fmt.Errorf("new error: %w", tt.originalErr)
			assert.Equal(t, Is(newErr, tt.isErr), tt.result)
		})
	}

	t.Run("is returns false on nil", func(t *testing.T) {
		assert.False(t, Is(fmt.Errorf("new error"), nil))
	})

}
