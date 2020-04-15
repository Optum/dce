package api

import (
	"testing"
)

// TestUsage tests the components of Usage
// This includes the APIs, step functions
func TestUsage(t *testing.T) {

	givenSystemIsEmpty(t)
	givenSystemHasUsageData(t)

	t.Run("Given system has usage data", func(t *testing.T) {

		t.Run("When listing all principal usage", func(t *testing.T) {
			t.Run("Then should get all principal usage records", func(t *testing.T) {

			})
		})
		t.Run("When listing lease usage with", func(t *testing.T) {
			t.Run("Then should get all lease usage records", func(t *testing.T) {

			})
		})
	})

	t.Run("Step Function", func(t *testing.T) {

	})
}
