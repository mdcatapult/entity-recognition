package html

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	snippet_reader "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader"
)

func TestHtmlToText(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    []snippet_reader.Value
		wantErr error
	}{
		{
			name: "empty body",
			args: args{
				r: bytes.NewBufferString(""),
			},
			want: []snippet_reader.Value{
				{Err: io.EOF},
			},
			wantErr: nil,
		},
		{
			name: "includes break",
			args: args{
				r: bytes.NewBufferString("  <body>  x<sup>2</sup> <strike>hello</strike><br/>dave</body>"),
			},
			want: wrapSnips([]*pb.Snippet{
				{
					Token:  "  x2 hello\ndave\n",
					Offset: 8,
					Xpath:  "/body",
				}}...),
			wantErr: nil,
		},
		{
			name: "only sends snippets at specific line break nodes",
			args: args{
				r: bytes.NewBufferString("<p>acetyl<emph>car</emph>nitine</p>"),
			},
			want: wrapSnips([]*pb.Snippet{
				{
					Token:  "acetylcarnitine\n",
					Offset: 3,
					Xpath:  "/p",
				},
			}...),
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Log(tt.name)
		vals := ReadSnippets(tt.args.r)

		i := 0
		for val := range vals {
			assert.EqualValues(t, tt.want[i], val)
			if val.Err != nil {
				break
			}
			i++
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

func wrapSnips(snips ...*pb.Snippet) []snippet_reader.Value {
	var values []snippet_reader.Value
	for _, snip := range snips {
		values = append(values, snippet_reader.Value{Snippet: snip})
	}
	values = append(values, snippet_reader.Value{Err: io.EOF})
	return values
}
