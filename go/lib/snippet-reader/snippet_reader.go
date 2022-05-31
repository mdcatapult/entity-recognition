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

package snippet_reader

import (
	"io"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
)

type Client interface {
	ReadSnippets(r io.Reader) <-chan Value
	ReadSnippetsWithCallback(r io.Reader, onSnippet func(*pb.Snippet) error) error
}

type Value struct {
	Snippet *pb.Snippet
	Err     error
}

func ReadChannelWithCallback(snipReaderValues <-chan Value, callback func(snippet *pb.Snippet) error) error {
	for readerValue := range snipReaderValues {
		if readerValue.Err == io.EOF {
			break
		} else if readerValue.Err != nil {
			return readerValue.Err
		}
		if err := callback(readerValue.Snippet); err != nil {
			return err
		}
	}
	return nil
}
