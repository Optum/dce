package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPrepareSNSMessageJSON(t *testing.T) {

	t.Run("should prepare an SNS message as JSON", func(t *testing.T) {
		obj := struct {
			Foo string `json:"foo"`
		}{
			Foo: "bar",
		}
		message, err := PrepareSNSMessageJSON(obj)

		require.Nil(t, err)
		require.Equal(t, message, `{"default":"{\"foo\":\"bar\"}","Body":"{\"foo\":\"bar\"}"}`)
	})

}
