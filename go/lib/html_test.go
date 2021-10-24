package lib

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

func TestHtmlToText(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    []*pb.Snippet
		wantErr error
	}{
		{
			name: "empty body",
			args: args{
				r: bytes.NewBufferString(""),
			},
			want:    []*pb.Snippet{},
			wantErr: nil,
		},
		{
			name: "includes break",
			args: args{
				r: bytes.NewBufferString("  <body>  x<sup>2</sup> <strike>hello</strike><br/>dave</body>"),
			},
			want: []*pb.Snippet{
				{
					Token:  "\n",
					Offset: 16,
					Xpath:  "/html/*[1]",
				},
				{
					Token:  "\n",
					Offset: 32,
					Xpath:  "/html/*[2]",
				},
				{
					Token:  "  x2 hello\ndave\n",
					Offset: 8,
					Xpath:  "/html",
				},
			},
			wantErr: nil,
		},
		{
			name: "only sends snippets at specific line break nodes",
			args: args{
				r: bytes.NewBufferString("<p>acetyl<emph>car</emph>nitine</p>"),
			},
			want: []*pb.Snippet{
				{
					Token:  "\n",
					Offset: 15,
					Xpath:  "/html/*[1]",
				},
				{
					Token:  "acetylcarnitine\n",
					Offset: 3,
					Xpath:  "/html",
				},
			},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Log(tt.name)
		i := 0
		gotSnips, gotErrs := HtmlToText(tt.args.r)
	Loop:
		for {
			select {
			case s := <-gotSnips:
				if i >= len(tt.want) {
					t.FailNow()
				}
				assert.EqualValues(t, tt.want[i], s)
				i++
			case err := <-gotErrs:
				assert.Equal(t, tt.wantErr, err)
				break Loop
			}
		}
	}
}

func Test_htmlStack_xpath(t *testing.T) {
	testStack := func(tags ...*htmlTag) htmlStack {
		stack := htmlStack{}
		for _, tag := range tags {
			stack.push(tag)
		}
		return stack
	}
	tests := []struct {
		stack    htmlStack
		expected string
	}{
		{
			expected: "/html/*[2]/*[4]/*[5]/*[3]",
			stack: testStack(&htmlTag{
				name:     "html",
				start:    0,
				children: 1,
			}, &htmlTag{
				name:     "body",
				start:    0,
				children: 3,
			}, &htmlTag{
				name:     "main",
				start:    0,
				children: 4,
			}, &htmlTag{
				name:     "article",
				start:    0,
				children: 2,
			}, &htmlTag{
				name:     "section",
				start:    0,
				children: 2,
			}),
		},
	}
	for _, tt := range tests {
		actual := tt.stack.xpath()
		assert.Equal(t, tt.expected, actual)
	}
}
