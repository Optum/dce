// Code generated by mockery v2.16.0. DO NOT EDIT.

package mocks

import (
	account "github.com/Optum/dce/pkg/account"

	arn "github.com/Optum/dce/pkg/arn"

	mock "github.com/stretchr/testify/mock"
)

// Servicer is an autogenerated mock type for the Servicer type
type Servicer struct {
	mock.Mock
}

// DeletePrincipalAccess provides a mock function with given fields: _a0
func (_m *Servicer) DeletePrincipalAccess(_a0 *account.Account) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*account.Account) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpsertPrincipalAccess provides a mock function with given fields: _a0
func (_m *Servicer) UpsertPrincipalAccess(_a0 *account.Account) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*account.Account) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ValidateAccess provides a mock function with given fields: role
func (_m *Servicer) ValidateAccess(role *arn.ARN) error {
	ret := _m.Called(role)

	var r0 error
	if rf, ok := ret.Get(0).(func(*arn.ARN) error); ok {
		r0 = rf(role)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewServicer interface {
	mock.TestingT
	Cleanup(func())
}

// NewServicer creates a new instance of Servicer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewServicer(t mockConstructorTestingTNewServicer) *Servicer {
	mock := &Servicer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}