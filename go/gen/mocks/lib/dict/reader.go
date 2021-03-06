// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"
	dict "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/dict"

	os "os"
)

// Reader is an autogenerated mock type for the Reader type
type Reader struct {
	mock.Mock
}

// Read provides a mock function with given fields: file
func (_m *Reader) Read(file *os.File) (chan dict.NerEntry, chan error) {
	ret := _m.Called(file)

	var r0 chan dict.NerEntry
	if rf, ok := ret.Get(0).(func(*os.File) chan dict.NerEntry); ok {
		r0 = rf(file)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(chan dict.NerEntry)
		}
	}

	var r1 chan error
	if rf, ok := ret.Get(1).(func(*os.File) chan error); ok {
		r1 = rf(file)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(chan error)
		}
	}

	return r0, r1
}
