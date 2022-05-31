/*
 * Copyright 2022 Medicines Discovery Catapult
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
