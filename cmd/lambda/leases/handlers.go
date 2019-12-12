package main

import (
	"fmt"

	"github.com/Optum/dce/pkg/common"
	"github.com/Optum/dce/pkg/db"
	"github.com/Optum/dce/pkg/usage"
)

// StepHandler - Handler for leases
type StepHandler func(context interface{}, lease *db.Lease) error

// StepConfiguration - Configuration for lease handlers
type StepConfiguration struct {
	name        string
	handler     StepHandler
	failOnError bool
}

// LeaseExecutionContext - Execution context for leases.
type LeaseExecutionContext struct {
	Dao      db.DBer
	SnsSvc   common.Notificationer
	UsageSvc usage.Service
}

// LeaseCommander - Interface for handling lease-related tasks
type LeaseCommander interface {
	AddStep(name string, handler StepHandler, failOnError bool) error
	Execute(context LeaseExecutionContext, lease *db.Lease) (bool, error)
	Errors() []error
}

// DefaultLeaseCommander - Default behavior
type DefaultLeaseCommander struct {
	taskConfiguration []StepConfiguration
	errorList         []error
}

// AddStep - Adds the tasks to the list of tasks to perform
func (tasker *DefaultLeaseCommander) AddStep(name string, handler StepHandler, failOnError bool) error {
	tasker.taskConfiguration = append(tasker.taskConfiguration, StepConfiguration{
		name:        name,
		handler:     handler,
		failOnError: failOnError,
	})
	return nil
}

// Execute - Performs all of the registered tasks
func (tasker *DefaultLeaseCommander) Execute(context LeaseExecutionContext, lease *db.Lease) (bool, error) {
	// Iterate through the tasks. Pay attention to the failOnError
	for _, t := range tasker.taskConfiguration {
		err := t.handler(context, lease)
		if err != nil {
			tasker.errorList = append(tasker.errorList, fmt.Errorf("%s: %s", t.name, err.Error()))
			if t.failOnError {
				return false, err
			}
		}
	}

	return (len(tasker.errorList) == 0), nil
}

// Errors gets the errors encountered during execution
func (tasker *DefaultLeaseCommander) Errors() []error {
	return tasker.errorList
}
