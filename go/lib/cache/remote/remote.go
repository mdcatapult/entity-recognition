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

package remote

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/gen/pb"
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
)

type Client interface {
	NewGetPipeline(size int) GetPipeline
	NewSetPipeline(size int) SetPipeline
	Ready() bool
}

type Pipeline interface {
	Size() int
}

type GetPipeline interface {
	Get(token *pb.Snippet)
	ExecGet(onResult func(*pb.Snippet, *cache.Lookup) error) error
	Pipeline
}

type SetPipeline interface {
	Set(key string, data []byte)
	ExecSet() error
	Pipeline
}
