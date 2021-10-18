package lib

import (
	"bytes"
	"container/list"
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
	disallowed bool
	disallowedDepth int
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
	if !s.disallowed {
		if _, ok := disallowedNodes[tag.name]; ok {
			s.disallowed = true
			s.disallowedDepth = s.Len()
		}
	}
}

func (s *htmlStack) pop() {
	e := s.Front()
	if s.disallowed && s.Len() == s.disallowedDepth {
		s.disallowed = false
		s.disallowedDepth = 0
	}
	s.Remove(e)
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
	buf := bytes.NewBuffer([]byte{})
	var currentSnippet *pb.Snippet

	// Function to send current snippet on the channel and
	// reset values necessary values.
	sendSnip := func() {
		bufBytes, err := buf.ReadBytes(0)
		if err != nil && err != io.EOF {
			errs <- err
			return
		}
		currentSnippet.Token = string(bufBytes)
		snips <- currentSnippet
		currentSnippet = nil
	}

Loop:
	for {
		htmlToken := htmlTokenizer.Next()
		switch htmlToken {
		case html.ErrorToken:
			// If we have a final snippet, send it!
			if currentSnippet != nil {
				sendSnip()
			}

			break Loop
		case html.TextToken:
			// Must read this first. Other read methods mutate the current token.
			htmlTokenBytes := htmlTokenizer.Text()

			// Only write to the buffer if we are not under any disallowed nodes.
			if !stack.disallowed {
				// Write the text to the buffer.
				buf.Write(htmlTokenBytes)

				// If the bytes are not composed only of whitespace.
				strippedBytes, nRemoved := StripLeft(htmlTokenBytes)
				if len(strippedBytes) > 0 {
					// increment the position by the amount of removed whitespace characters.
					position += uint32(nRemoved)

					// Instantiate a new snippet if it is nil (i.e. we've just sent
					// the last one).
					if currentSnippet == nil {
						currentSnippet = &pb.Snippet{
							Token:  "",
							Offset: position,
						}
					}
				}
			}

			position += uint32(len(htmlTokenBytes))
		case html.StartTagToken:
			// Must read this first. Other read methods mutate the current token.
			htmlTokenBytes := htmlTokenizer.Raw()

			stackWasPreviouslyDisallowed := stack.disallowed

			// Push the tag onto the stack.
			tn, _ := htmlTokenizer.TagName()
			stack.push(htmlTag{name: string(tn), start: position})

			// Send a snippet if we are about to enter a disallowed tree.
			if stack.disallowed && !stackWasPreviouslyDisallowed && currentSnippet != nil {
				sendSnip()
			}

			position += uint32(len(htmlTokenBytes))
		case html.EndTagToken:
			// Must read this first. Other read methods mutate the current token.
			htmlTokenBytes := htmlTokenizer.Raw()

			// If we are at a linebreak node, not in a disallowed DOM tree, and the current snippet is not nil,
			// write a newline to the snippet and send it.
			tn, _ := htmlTokenizer.TagName()
			if _, breakLine := linebreakNodes[string(tn)]; breakLine && !stack.disallowed && currentSnippet != nil {
				buf.Write([]byte{'\n'})
				sendSnip()
			}

			// Remove this tag from the stack.
			stack.pop()
			position += uint32(len(htmlTokenBytes))
		case html.SelfClosingTagToken:
			// Must read this first. Other read methods mutate the current token.
			htmlTokenBytes := htmlTokenizer.Raw()

			// If we are at a linebreak node, not in a disallowed DOM tree, and the current snippet is not nil,
			// write a newline to the snippet and send it.
			tn, _ := htmlTokenizer.TagName()
			if _, breakLine := linebreakNodes[string(tn)]; breakLine && !stack.disallowed && currentSnippet != nil {
				buf.Write([]byte{'\n'})
				sendSnip()
			}
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