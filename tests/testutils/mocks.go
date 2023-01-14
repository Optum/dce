package testutils

import "github.com/stretchr/testify/mock"

// ReplaceMock allows you to mock out a method,
// overriding any previous expectations for that same method.
//
// This is useful if you want to setup a generic stub object
// to reuse among multiple tests, but then tweak expectations
// for some of the tests.
//
// Note that you need to pass the underlying mock object to this method,
// as `mock.Mock` does not expose any useful interfaces
// eg.
//
//	ReplaceMock(&myMock.Mock, "DoAThing", mock.Anything).Return(nil)
func ReplaceMock(m *mock.Mock, methodName string, arguments ...interface{}) *mock.Call {
	// Recreated the list of expected calls,
	// excluding calls for this method
	var expectedCalls []*mock.Call
	for _, call := range m.ExpectedCalls {
		if call.Method != methodName {
			expectedCalls = append(expectedCalls, call)
		}
	}
	m.ExpectedCalls = expectedCalls

	return m.On(methodName, arguments...)
}
