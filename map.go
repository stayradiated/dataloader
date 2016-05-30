package dataloader

import (
	"sync"
)

type Map struct {
	x *sync.Mutex
	m map[interface{}]interface{}
}

func newMap() *Map {
	return &Map{
		x: &sync.Mutex{},
		m: make(map[interface{}]interface{}),
	}
}

func (m *Map) Get(key interface{}) interface{} {
	m.x.Lock()
	defer m.x.Unlock()
	return m.m[key]
}

func (m *Map) Set(key interface{}, value interface{}) {
	m.x.Lock()
	defer m.x.Unlock()
	m.m[key] = value
}

func (m Map) Delete(key interface{}) {
	m.x.Lock()
	defer m.x.Unlock()
	delete(m.m, key)
}

func (m *Map) Clear() {
	m.x.Lock()
	defer m.x.Unlock()
	m.m = make(map[interface{}]interface{})
}
