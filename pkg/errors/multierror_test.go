package errors

import (
	"errors"
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

	t.Run("formatted multierror", func(t *testing.T) {
			intErr := NewInternalServer("There's an internal error!", errors.New("Low Level Error Info."))
			mErr := NewMultiError("We got a multi error!", []error{
				intErr,
			})
			mErrFmtd := fmt.Sprintf("%+v", mErr)

			// Check that we're getting verbose formatting from nested errors
			assert.Contains(t, mErrFmtd, "We got a multi error!")
			assert.Contains(t, mErrFmtd, "There's an internal error!")
			assert.Contains(t, mErrFmtd, "Low Level Error Info.")
	})

}
