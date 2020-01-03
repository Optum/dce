package tasks

import (
	"context"
	"fmt"

	"github.com/Optum/dce/pkg/db"
)

// StepHandler - Handler for leases
type StepHandler func(ctx context.Context, lease *db.Lease) error

// StepConfiguration - Configuration for lease handlers
type StepConfiguration struct {
	name        string
	handler     StepHandler
	failOnError bool
}

// MultiStepHandler - Default behavior
type MultiStepHandler struct {
	TaskConfiguration []StepConfiguration
	ErrorList         []error
}

// AddStep - Adds the tasks to the list of tasks to perform
func (m *MultiStepHandler) AddStep(name string, handler StepHandler, failOnError bool) error {
	m.TaskConfiguration = append(m.TaskConfiguration, StepConfiguration{
		name:        name,
		handler:     handler,
		failOnError: failOnError,
	})
	return nil
}

// Execute - Performs all of the registered tasks
func (m *MultiStepHandler) Execute(ctx context.Context, lease *db.Lease) (bool, error) {
	// Iterate through the tasks. Pay attention to the failOnError
	for _, t := range m.TaskConfiguration {
		err := t.handler(ctx, lease)
		if err != nil {
			m.ErrorList = append(m.ErrorList, fmt.Errorf("%s: %s", t.name, err.Error()))
			if t.failOnError {
				return false, err
			}
		}
	}

	return (len(m.ErrorList) == 0), nil
}

// Errors gets the errors encountered during execution
func (m *MultiStepHandler) Errors() []error {
	return m.ErrorList
}
