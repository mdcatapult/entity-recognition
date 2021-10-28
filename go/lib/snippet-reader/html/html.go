package html

import (
	"io"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	snippet_reader "gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/snippet-reader"
	"golang.org/x/net/html"
)

var disallowedNodes = map[string]struct{}{
	"area":     {},
	"audio":    {},
	"link":     {},
	"meta":     {},
	"noscript": {},
	"script":   {},
	"source":   {},
	"style":    {},
	"input":    {},
	"textarea": {},
	"video":    {},
}

var nonBreakingNodes = map[string]struct{}{
	"span":   {},
	"sub":    {},
	"sup":    {},
	"b":      {},
	"del":    {},
	"i":      {},
	"ins":    {},
	"mark":   {},
	"q":      {},
	"s":      {},
	"strike": {},
	"strong": {},
	"u":      {},
	"big":    {},
	"small":  {},
	"a":      {},
	"emph":   {},
}

type SnippetReader struct{}

func (SnippetReader) ReadSnippets(r io.Reader) <-chan snippet_reader.Value {
	return ReadSnippets(r)
}
func (s SnippetReader) ReadSnippetsWithCallback(r io.Reader, onSnippet func(*pb.Snippet) error) error {
	snips := ReadSnippets(r)
	return snippet_reader.ReadChannelWithCallback(snips, onSnippet)
}

// ReadSnippets is a convenience function so that the caller doesn't need to instantiate
// a channel.
func ReadSnippets(r io.Reader) <-chan snippet_reader.Value {
	snips := make(chan snippet_reader.Value)
	go htmlToText(r, snips)
	return snips
}

func ReadSnippetsWithCallback(r io.Reader, onSnippet func(*pb.Snippet) error) error {
	snips := ReadSnippets(r)
	return snippet_reader.ReadChannelWithCallback(snips, onSnippet)
}

// htmlToText Uses the html parser from the golang standard lib to get sequential
// tokens. We keep track of the current html tag so we know whether to include the
// text or not. When we reach an end tag (i.e. </p>), send the snippet to the snips
// channel. Additionally, add line breaks where appropriate.
func htmlToText(r io.Reader, snips chan snippet_reader.Value) {
	htmlTokenizer := html.NewTokenizer(r)
	var position uint32
	var stack htmlStack

	stackPopCallback := func(tag *htmlTag) error {
		if len(tag.innerText) > 0 {
			tag.innerText = append(tag.innerText, '\n')
			snips <- snippet_reader.Value{
				Snippet: &pb.Snippet{
					Text:  string(tag.innerText),
					Offset: tag.start,
					Xpath:  tag.xpath,
				},
			}
		}
		return nil
	}

	for {
		htmlToken := htmlTokenizer.Next()
		switch htmlToken {
		case html.ErrorToken:
			// The html tokenizer returns an io.EOF when finished.
			snips <- snippet_reader.Value{Err: htmlTokenizer.Err()}
		case html.TextToken:
			htmlTokenBytes := htmlTokenizer.Text()

			// Only write to the buffer if we are not under any disallowed nodes.
			if !stack.disallowed {
				stack.collectText(htmlTokenBytes)
			}

			position += uint32(len(htmlTokenBytes))
		case html.StartTagToken:
			// Must read this first. Other read methods mutate the current token.
			htmlTokenBytes := htmlTokenizer.Raw()

			// Push the tag onto the stack.
			tn, _ := htmlTokenizer.TagName()
			position += uint32(len(htmlTokenBytes))
			stack.push(&htmlTag{name: string(tn), start: position})

		case html.EndTagToken:
			// Must read this first. Other read methods mutate the current token.
			htmlTokenBytes := htmlTokenizer.Raw()

			// Remove this tag from the stack.
			if err := stack.pop(stackPopCallback); err != nil {
				snips <- snippet_reader.Value{Err: err}
				return
			}

			position += uint32(len(htmlTokenBytes))
		case html.SelfClosingTagToken:
			// Must read this first. Other read methods mutate the current token.
			htmlTokenBytes := htmlTokenizer.Raw()

			// If we are at a linebreak node, not in a disallowed DOM tree, and the current snippet is not nil,
			// write a newline to the snippet and send it.
			tn, _ := htmlTokenizer.TagName()
			if string(tn) == "br" {
				stack.collectText([]byte{'\n'})
			}
			stack.push(&htmlTag{name: string(tn), start: position})
			_ = stack.pop(func(tag *htmlTag) error { return nil })
			position += uint32(len(htmlTokenBytes))
		default:
			htmlTokenBytes := htmlTokenizer.Raw()
			position += uint32(len(htmlTokenBytes))
		}
	}
}
