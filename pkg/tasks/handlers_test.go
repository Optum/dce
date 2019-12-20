package tasks

import (
	"context"
	"errors"
	"testing"

	"github.com/Optum/dce/pkg/db"
	"github.com/stretchr/testify/assert"
)

func TestLeaseHandlers(t *testing.T) {

	t.Run("Handlers with failOnError should fail", func(t *testing.T) {

		lease := &db.Lease{
			AccountID: "123456789012",
		}

		ctx := context.Background()

		leaseCreator := &MultiStepHandler{}

		leaseCreator.AddStep("Step 1", Step1, true)
		leaseCreator.AddStep("Step 2", Step2, true)
		_, err := leaseCreator.Execute(ctx, lease)

		assert.NotNil(t, err)
		assert.Equal(t, db.Inactive, lease.LeaseStatus)
		assert.Equal(t, 1, len(leaseCreator.Errors()))

	})

	t.Run("Handlers with failOnError set to false should continue", func(t *testing.T) {

		lease := &db.Lease{
			AccountID: "123456789012",
		}

		ctx := context.Background()

		leaseCreator := &MultiStepHandler{}

		leaseCreator.AddStep("Step 1", Step1, false)
		leaseCreator.AddStep("Step 2", Step2, true)
		success, err := leaseCreator.Execute(ctx, lease)

		// Overall execution is failed, but there is no overall error
		assert.False(t, success)
		assert.Nil(t, err)
		assert.Equal(t, db.Active, lease.LeaseStatus)
		// The error is still there. It's just that the step was configured NOT
		// to fail on the error
		assert.Equal(t, 1, len(leaseCreator.Errors()))

	})

	t.Run("Handlers with failOnError set to false should continue", func(t *testing.T) {

		lease := &db.Lease{
			AccountID: "123456789012",
		}

		ctx := context.Background()

		leaseCreator := &MultiStepHandler{}

		leaseCreator.AddStep("Step 2", Step2, true)
		success, err := leaseCreator.Execute(ctx, lease)

		// Only one successful task, should be true.
		assert.True(t, success)
		assert.Nil(t, err)
		assert.Equal(t, db.Active, lease.LeaseStatus)
		// The error is still there. It's just that the step was configured NOT
		// to fail on the error
		assert.Equal(t, 0, len(leaseCreator.Errors()))

	})

}

func Step1(ctx context.Context, lease *db.Lease) error {
	lease.LeaseStatus = db.Inactive
	return errors.New("error while performing step")
}

func Step2(ctx context.Context, lease *db.Lease) error {
	lease.LeaseStatus = db.Active
	return nil
}
