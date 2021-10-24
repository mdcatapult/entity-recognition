package lib

import (
	"bytes"
	"container/list"
	"fmt"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"golang.org/x/net/html"
	"io"
	"unicode"
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
	"span": {},
	"sub": {},
	"sup": {},
	"b": {},
	"del": {},
	"i": {},
	"ins": {},
	"mark": {},
	"q": {},
	"s": {},
	"strike": {},
	"strong": {},
	"u": {},
	"big": {},
	"small": {},
	"a": {},
}

type htmlStack struct {
	*list.List
	disallowed bool
	disallowedDepth int
	appendMode      bool
	appendModeTag   *htmlTag
	appendModeDepth int
}

type htmlTag struct {
	name  string
	start uint32
	children int
	innerText []byte
	xpath string
}

func (s *htmlStack) push(tag *htmlTag) {
	if s.List == nil {
		s.List = list.New()
	}

	if front := s.List.Front(); front != nil {
		front.Value.(*htmlTag).children++
	}


	if !s.appendMode {
		if _, ok := nonBreakingNodes[tag.name]; ok {
			s.appendMode = true
			s.appendModeDepth = s.Len()+1
			s.appendModeTag = s.Front().Value.(*htmlTag)
		}
	}

	s.PushFront(tag)
	tag.xpath = s.xpath()

	if !s.disallowed {
		if _, ok := disallowedNodes[tag.name]; ok {
			s.disallowed = true
			s.disallowedDepth = s.Len()
		}
	}
}

func (s *htmlStack) collectText(text []byte) {
	if s.List == nil {
		s.List = list.New()
	}
	if s != nil && s.Front() != nil {
		var tag *htmlTag
		if s.appendMode {
			tag = s.appendModeTag
		} else {
			tag = s.Front().Value.(*htmlTag)
		}
		tag.innerText = append(tag.innerText, text...)
	}
}

func (s *htmlStack) pop(callback func(tag *htmlTag) error) error {
	e := s.Front()
	if e == nil {
		return io.EOF
	}
	if s.disallowed && s.Len() == s.disallowedDepth {
		s.disallowed = false
		s.disallowedDepth = 0
	}
	if s.appendMode && s.Len() == s.appendModeDepth {
		s.appendMode = false
		s.appendModeDepth = 0
		s.appendModeTag = nil
	}
	tag := e.Value.(*htmlTag)

	s.Remove(e)
	if !s.appendMode {
		return callback(tag)
	}

	return nil
}

func (s *htmlStack) xpath() string {
	element := s.List.Back()
	var path = "/html"
	for {
		if element.Next() != nil {
			path = fmt.Sprintf("%s/*[%d]", path, element.Next().Value.(*htmlTag).children)
		}
		if element.Prev() != nil {
			element = element.Prev()
		} else {
			break
		}
	}
	return path
}

func HtmlToTextWithCallback(r io.Reader, onSnippet func(*pb.Snippet) error) error {
	snips, errs := HtmlToText(r)

Loop:
	for {
		select {
		case s := <-snips:
			if err := onSnippet(s); err != nil {
				return err
			}
		case err := <-errs:
			if err != nil {
				return err
			}
			break Loop
		}
	}
	return nil
}

// HtmlToText is a convenience function so that the caller doesn't need to instantiate
// a channel.
func HtmlToText(r io.Reader) (<-chan *pb.Snippet, <-chan error) {
	snips := make(chan *pb.Snippet)
	errs := make(chan error)
	go htmlToText(r, snips, errs)
	return snips, errs
}

// htmlToText Uses the html parser from the golang standard lib to get sequential
// tokens. We keep track of the current html tag so we know whether to include the
// text or not. When we reach an end tag (i.e. </p>), send the snippet to the snips
// channel. Additionally, add line breaks where appropriate.
func htmlToText(r io.Reader, snips chan *pb.Snippet, errs chan error) {
	htmlTokenizer := html.NewTokenizer(r)
	var position uint32
	var stack htmlStack

	stackPopCallback := func(tag *htmlTag) error {
		tag.innerText = append(tag.innerText, '\n')
		snips <- &pb.Snippet{
			Token:  string(tag.innerText),
			Offset: tag.start,
			Xpath:  tag.xpath,
		}
		return nil
	}

Loop:
	for {
		htmlToken := htmlTokenizer.Next()
		switch htmlToken {
		case html.ErrorToken:
			// If we have a final snippet, send it!
			for {
				err := stack.pop(stackPopCallback)
				if err == io.EOF {
					break
				} else if err != nil {
					errs <- err
					return
				}
			}

			break Loop
		case html.TextToken:
			// Must read this first. Other read methods mutate the current token.
			htmlTokenBytes := htmlTokenizer.Text()

			// Only write to the buffer if we are not under any disallowed nodes.
			if !stack.disallowed {
				// Write the text to the buffer.
				stack.collectText(htmlTokenBytes)

				// If the bytes are not composed only of whitespace.
				//strippedBytes, nRemoved := StripLeft(htmlTokenBytes)
				//if len(strippedBytes) > 0 {
				//	// increment the position by the amount of removed whitespace characters.
				//	position += uint32(nRemoved)
				//}
			}

			position += uint32(len(htmlTokenBytes))
		case html.StartTagToken:
			// Must read this first. Other read methods mutate the current token.
			htmlTokenBytes := htmlTokenizer.Raw()

			// Push the tag onto the stack.
			tn, _ := htmlTokenizer.TagName()
			stack.push(&htmlTag{name: string(tn), start: position})

			position += uint32(len(htmlTokenBytes))
		case html.EndTagToken:
			// Must read this first. Other read methods mutate the current token.
			htmlTokenBytes := htmlTokenizer.Raw()

			// Remove this tag from the stack.
			if err := stack.pop(stackPopCallback); err != nil && err != io.EOF {
				errs <- err
				return
			}

			position += uint32(len(htmlTokenBytes))
		case html.SelfClosingTagToken:
			// Must read this first. Other read methods mutate the current token.
			htmlTokenBytes := htmlTokenizer.Raw()

			// If we are at a linebreak node, not in a disallowed DOM tree, and the current snippet is not nil,
			// write a newline to the snippet and send it.
			tn, _ := htmlTokenizer.TagName()
			stack.push(&htmlTag{name: string(tn), start: position})
			_ = stack.pop(func(tag *htmlTag) error {return nil})
			position += uint32(len(htmlTokenBytes))
		default:
			htmlTokenBytes := htmlTokenizer.Raw()
			position += uint32(len(htmlTokenBytes))
		}
	}
	if err := htmlTokenizer.Err(); err != io.EOF {
		errs <- err
	}
	errs <- nil
}

// StripLeft returns a byte slice equal to the given byte slice with all leading whitespace characters removed,
// and an integer indicating how many characters were stripped.
func StripLeft(b []byte) (strippedBytes []byte, nStripped int) {
	left := 0
	started := false
	return bytes.Map(func(r rune) rune {
		if !started && unicode.IsSpace(r) {
			left++
			return -1
		}
		started = true
		return r
	}, b), left
}