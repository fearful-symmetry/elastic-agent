// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Code generated by mockery v2.20.0. DO NOT EDIT.

// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	composable "github.com/elastic/elastic-agent/internal/pkg/core/composable"
)

// FetchContextProvider is an autogenerated mock type for the FetchContextProvider type
type FetchContextProvider struct {
	mock.Mock
}

type FetchContextProvider_Expecter struct {
	mock *mock.Mock
}

func (_m *FetchContextProvider) EXPECT() *FetchContextProvider_Expecter {
	return &FetchContextProvider_Expecter{mock: &_m.Mock}
}

// Fetch provides a mock function with given fields: _a0
func (_m *FetchContextProvider) Fetch(_a0 string) (string, bool) {
	ret := _m.Called(_a0)

	var r0 string
	var r1 bool
	if rf, ok := ret.Get(0).(func(string) (string, bool)); ok {
		return rf(_a0)
	}
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(string) bool); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Get(1).(bool)
	}

	return r0, r1
}

// FetchContextProvider_Fetch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Fetch'
type FetchContextProvider_Fetch_Call struct {
	*mock.Call
}

// Fetch is a helper method to define mock.On call
//   - _a0 string
func (_e *FetchContextProvider_Expecter) Fetch(_a0 interface{}) *FetchContextProvider_Fetch_Call {
	return &FetchContextProvider_Fetch_Call{Call: _e.mock.On("Fetch", _a0)}
}

func (_c *FetchContextProvider_Fetch_Call) Run(run func(_a0 string)) *FetchContextProvider_Fetch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *FetchContextProvider_Fetch_Call) Return(_a0 string, _a1 bool) *FetchContextProvider_Fetch_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *FetchContextProvider_Fetch_Call) RunAndReturn(run func(string) (string, bool)) *FetchContextProvider_Fetch_Call {
	_c.Call.Return(run)
	return _c
}

// Run provides a mock function with given fields: _a0
func (_m *FetchContextProvider) Run(_a0 composable.ContextProviderComm) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(composable.ContextProviderComm) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// FetchContextProvider_Run_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Run'
type FetchContextProvider_Run_Call struct {
	*mock.Call
}

// Run is a helper method to define mock.On call
//   - _a0 composable.ContextProviderComm
func (_e *FetchContextProvider_Expecter) Run(_a0 interface{}) *FetchContextProvider_Run_Call {
	return &FetchContextProvider_Run_Call{Call: _e.mock.On("Run", _a0)}
}

func (_c *FetchContextProvider_Run_Call) Run(run func(_a0 composable.ContextProviderComm)) *FetchContextProvider_Run_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(composable.ContextProviderComm))
	})
	return _c
}

func (_c *FetchContextProvider_Run_Call) Return(_a0 error) *FetchContextProvider_Run_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *FetchContextProvider_Run_Call) RunAndReturn(run func(composable.ContextProviderComm) error) *FetchContextProvider_Run_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewFetchContextProvider interface {
	mock.TestingT
	Cleanup(func())
}

// NewFetchContextProvider creates a new instance of FetchContextProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewFetchContextProvider(t mockConstructorTestingTNewFetchContextProvider) *FetchContextProvider {
	mock := &FetchContextProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}