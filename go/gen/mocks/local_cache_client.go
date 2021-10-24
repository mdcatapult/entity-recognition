// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import (
	cache "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"

	mock "github.com/stretchr/testify/mock"
)

// LocalCacheClient is an autogenerated mock type for the LocalCacheClient type
type LocalCacheClient struct {
	mock.Mock
}

// Delete provides a mock function with given fields: key
func (_m *LocalCacheClient) Delete(key string) {
	_m.Called(key)
}

// Get provides a mock function with given fields: key
func (_m *LocalCacheClient) Get(key string) *cache.Lookup {
	ret := _m.Called(key)

	var r0 *cache.Lookup
	if rf, ok := ret.Get(0).(func(string) *cache.Lookup); ok {
		r0 = rf(key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*cache.Lookup)
		}
	}

	return r0
}

// Set provides a mock function with given fields: key, lookup
func (_m *LocalCacheClient) Set(key string, lookup *cache.Lookup) {
	_m.Called(key, lookup)
}
