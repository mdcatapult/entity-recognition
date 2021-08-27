package local

import (
	"gitlab.mdcatapult.io/informatics/software-engineering/entity-recognition/go/lib/cache"
	"sync"
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
	mut *sync.RWMutex
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
