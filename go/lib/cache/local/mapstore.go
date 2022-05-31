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

package local

import (
	"sync"

	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
)

func New() Client {
	return &local{
		store: make(map[string]*cache.Lookup),
		mut:   &sync.RWMutex{},
	}
}

type Client interface {
	Get(key string) *cache.Lookup
	Set(key string, lookup *cache.Lookup)
	Delete(key string)
}

type local struct {
	store map[string]*cache.Lookup
	mut   *sync.RWMutex
}

func (l *local) Get(key string) *cache.Lookup {
	l.mut.RLock()
	defer l.mut.RUnlock()

	lookup, ok := l.store[key]
	if !ok {
		return nil
	}

	return lookup
}

func (l *local) Set(key string, lookup *cache.Lookup) {
	l.mut.Lock()
	defer l.mut.Unlock()

	l.store[key] = lookup
}

func (l *local) Delete(key string) {
	l.mut.Lock()
	defer l.mut.Unlock()

	delete(l.store, key)
}
