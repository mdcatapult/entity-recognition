package lib

import (
	"bytes"
	"container/list"
	"io"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"golang.org/x/net/html"
)

var disallowedNodes = map[string]struct{}{
	"area":     {},
	"audio":    {},
	"head":     {},
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

var linebreakNodes = map[string]struct{}{
	"ol":         {},
	"ul":         {},
	"li":         {},
	"table":      {},
	"tbody":      {},
	"tr":         {},
	"th":         {},
	"br":         {},
	"h1":         {},
	"h2":         {},
	"h3":         {},
	"h4":         {},
	"h5":         {},
	"h6":         {},
	"p":          {},
	"section":    {},
	"header":     {},
	"article":    {},
	"aside":      {},
	"summary":    {},
	"figure":     {},
	"figcaption": {},
	"footer":     {},
	"nav":        {},
}

type htmlStack struct {
	*list.List
}

type htmlTag struct {
	name  string
	start uint32
}

func (s *htmlStack) push(tag htmlTag) {
	if s.List == nil {
		s.List = list.New()
	}
	s.PushFront(tag)
}

func (s *htmlStack) top() htmlTag {
	if s.List == nil {
		s.List = list.New()
	}
	e := s.Front()
	if e == nil {
		return htmlTag{}
	}
	return e.Value.(htmlTag)
}

func (s *htmlStack) pop() {
	e := s.Front()
	s.Remove(e)
}

func HtmlToTextWithCallback()

// HtmlToText is a convenience function so that the caller doesn't need to instantiate
// a channel.
func HtmlToText(r io.Reader) (<-chan *pb.Snippet, error) {
	var snips chan *pb.Snippet
	return snips, htmlToText(r, snips)
}

// htmlToText Uses the html parser from the golang standard lib to get sequential
// tokens. We keep track of the current html tag so we know whether to include the
// text or not. When we reach an end tag (i.e. </p>), send the snippet to the snips
// channel. Additionally, add line breaks where appropriate.
func htmlToText(r io.Reader, snips chan *pb.Snippet) error {
	htmlTokenizer := html.NewTokenizer(r)
	var position uint32
	var stack htmlStack
	buf := bytes.NewBuffer([]byte{})

Loop:
	for {
		htmlToken := htmlTokenizer.Next()
		switch htmlToken {
		case html.ErrorToken:
			break Loop
		case html.TextToken:
			htmlTokenBytes := htmlTokenizer.Text()
			if _, disallowed := disallowedNodes[stack.top().name]; !disallowed {
				buf.Write(htmlTokenBytes)
			}
			position += uint32(len(htmlTokenBytes))
		case html.StartTagToken:
			htmlTokenBytes := htmlTokenizer.Raw()
			tn, _ := htmlTokenizer.TagName()
			stack.push(htmlTag{name: string(tn), start: position})
			position += uint32(len(html.UnescapeString(string(htmlTokenBytes))))
		case html.EndTagToken:
			htmlTokenBytes := htmlTokenizer.Raw()
			tn, _ := htmlTokenizer.TagName()
			if _, disallowed := disallowedNodes[stack.top().name]; !disallowed && string(tn) == stack.top().name {
				if _, breakLine := linebreakNodes[stack.top().name]; breakLine {
					buf.Write([]byte{'\n'})
				}
				bufferBytes, err := buf.ReadBytes(0)
				if err != nil && err != io.EOF {
					return err
				}
				snips <- &pb.Snippet{
					Token:  string(bufferBytes),
					Offset: stack.top().start,
				}
			}
			position += uint32(len(html.UnescapeString(string(htmlTokenBytes))))
			stack.pop()
		case html.SelfClosingTagToken:
			htmlTokenBytes := htmlTokenizer.Raw()
			tn, _ := htmlTokenizer.TagName()
			if _, breakLine := linebreakNodes[string(tn)]; breakLine {
				buf.Write([]byte{'\n'})
			}
			position += uint32(len(html.UnescapeString(string(htmlTokenBytes))))
		default:
			htmlTokenBytes := htmlTokenizer.Raw()
			position += uint32(len(html.UnescapeString(string(htmlTokenBytes))))
		}
	}
	if err := htmlTokenizer.Err(); err != io.EOF {
		return err
	}
	return nil
}
