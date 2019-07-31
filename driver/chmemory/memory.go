package chmemory

import (
	"sync"
	"sync/atomic"

	"github.com/coyove/ch/driver"
)

func NewNode(name string, weight int64) *driver.Node {
	return &driver.Node{
		KV:     &Storage{},
		Name:   name,
		Weight: weight,
	}
}

type Storage struct {
	kv    sync.Map
	count int64
}

func (s *Storage) Put(k string, v []byte) error {
	s.kv.Store(k, v)
	atomic.AddInt64(&s.count, 1)
	return nil
}

func (s *Storage) Get(k string) ([]byte, error) {
	v, ok := s.kv.Load(k)
	if ok {
		return v.([]byte), nil
	}
	return nil, driver.ErrKeyNotFound
}

func (s *Storage) Delete(k string) error {
	s.kv.Delete(k)
	atomic.AddInt64(&s.count, -1)
	return nil
}

func (s *Storage) Stat() driver.Stat {
	return driver.Stat{
		ObjectCount: s.count,
	}
}