package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestJwtToken(t *testing.T) {
	t.Run("getTID", func(t *testing.T) {

		t.Run("should parse a token", func(t *testing.T) {
			token := "STUB.eyJ0aWQiOiAiU1RVQl9USUQifQ.STUB"
			res, err := getTID(token)
			require.Nil(t, err)

			require.Equal(t, "STUB_TID", res)
		})

	})
}
