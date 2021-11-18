// Code generated by mockery v2.9.4. DO NOT EDIT.

package mocks

import (
	io "io"

	mock "github.com/stretchr/testify/mock"
	pb "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	snippet_reader "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

// ReadSnippets provides a mock function with given fields: r
func (_m *Client) ReadSnippets(r io.Reader) <-chan snippet_reader.Value {
	ret := _m.Called(r)

	var r0 <-chan snippet_reader.Value
	if rf, ok := ret.Get(0).(func(io.Reader) <-chan snippet_reader.Value); ok {
		r0 = rf(r)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan snippet_reader.Value)
		}
	}

	return r0
}

// ReadSnippetsWithCallback provides a mock function with given fields: r, onSnippet
func (_m *Client) ReadSnippetsWithCallback(r io.Reader, onSnippet func(*pb.Snippet) error) error {
	ret := _m.Called(r, onSnippet)

	var r0 error
	if rf, ok := ret.Get(0).(func(io.Reader, func(*pb.Snippet) error) error); ok {
		r0 = rf(r, onSnippet)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}