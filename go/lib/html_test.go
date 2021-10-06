package lib

import (
	"bytes"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"io"
	"testing"
)

func TestHtmlToText(t *testing.T) {
	type args struct {
		r         io.Reader
		onSnippet func(snippet *pb.Snippet) error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "",
			args: args{
				r: bytes.NewBufferString(""),
				onSnippet: func(snippet *pb.Snippet) error {

				},
			},
		},
	}
	for _, tt := range tests {

	}
}
