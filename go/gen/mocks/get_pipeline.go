// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"
	pb "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	db "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
)

// GetPipeline is an autogenerated mock type for the GetPipeline type
type GetPipeline struct {
	mock.Mock
}

// ExecGet provides a mock function with given fields: onResult
func (_m *GetPipeline) ExecGet(onResult func(*pb.Snippet, *db.Lookup) error) error {
	ret := _m.Called(onResult)

	var r0 error
	if rf, ok := ret.Get(0).(func(func(*pb.Snippet, *db.Lookup) error) error); ok {
		r0 = rf(onResult)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Get provides a mock function with given fields: token
func (_m *GetPipeline) Get(token *pb.Snippet) {
	_m.Called(token)
}

// Size provides a mock function with given fields:
func (_m *GetPipeline) Size() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}
