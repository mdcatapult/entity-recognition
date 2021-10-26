package html

import (
	"container/list"
	"fmt"
)

type htmlStack struct {
	*list.List
	disallowed      bool
	disallowedDepth int
	appendMode      bool
	appendModeTag   *htmlTag
	appendModeDepth int
}

type htmlTag struct {
	name      string
	start     uint32
	children  int
	innerText []byte
	xpath     string
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
			s.appendModeDepth = s.Len() + 1
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
	if s.List == nil {
		s.List = list.New()
	}

	e := s.Front()
	if e == nil {
		return nil
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
	return callback(tag)
}

func (s *htmlStack) xpath() string {
	path := "/"
	element := s.List.Back()
	if element != nil {
		path += element.Value.(*htmlTag).name
	}
	for {
		if element.Next() != nil {
			path += fmt.Sprintf("/*[%d]", element.Next().Value.(*htmlTag).children)
		}
		if element.Prev() != nil {
			element = element.Prev()
		} else {
			break
		}
	}
	return path
}
