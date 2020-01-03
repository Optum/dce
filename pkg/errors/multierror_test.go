package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiError(t *testing.T) {

	t.Run("new multierror", func(t *testing.T) {

		err1 := fmt.Errorf("err1")
		err2 := fmt.Errorf("err2")

		errs := NewMultiError("many errors", []error{err1, err2})

		assert.Equal(t, errs.Error(), "many errors: err1; err2")
	})

}
