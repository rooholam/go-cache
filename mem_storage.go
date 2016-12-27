package cache

import (
	"sync"
	"time"
)

const (
	STORAGE_TYPE_MEMORY = iota
	STORAGE_TYPE_REDIS
)

type Storage interface {
	Set(string, Item)
	Get(string) (Item, bool)
	GetObject(string, interface{}) (Item, bool)
	Del(key string)
	Flush()
	Type() int

	Lock()
	Unlock()
	RLock()
	RUnlock()
}

type memoryStorage struct {
	items   map[string]Item
	mutex   sync.RWMutex
	janitor *janitor
}

func (s *memoryStorage) Get(key string) (Item, bool) {
	item, found := s.items[key]
	return item, found
}
func (s *memoryStorage) GetObject(key string, o interface{}) (Item, bool) {
	return s.Get(key)
}

func (s *memoryStorage) Set(key string, item Item) {
	s.items[key] = item
}

func (s *memoryStorage) Del(key string) {
	delete(s.items, key)
}

func (s *memoryStorage) DeleteExpired() {
	now := time.Now().UnixNano()
	s.Lock()
	for k, v := range s.items {
		if v.Expiration > 0 && now > v.Expiration {
			s.Del(k)
		}
	}
	s.Unlock()
}

func (s *memoryStorage) Flush() {
	s.Lock()
	s.items = map[string]Item{}
	s.Unlock()
}

func (s *memoryStorage) Lock() {
	s.mutex.Lock()
}
func (s *memoryStorage) Unlock() {
	s.mutex.Unlock()
}

func (s *memoryStorage) RLock() {
	s.mutex.RLock()
}
func (s *memoryStorage) RUnlock() {
	s.mutex.RUnlock()
}

func (s *memoryStorage) Type() int {
	return STORAGE_TYPE_MEMORY
}

func MemoryStorage() *memoryStorage {
	mem := memoryStorage{
		items:make(map[string]Item),
	}
	return &mem
}

type janitor struct {
	Interval time.Duration
	stop     chan bool
}

func (j *janitor) Run(s *memoryStorage) {
	j.stop = make(chan bool)
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			s.DeleteExpired()
		case <-j.stop:
			ticker.Stop()
			return
		}
	}
}

func stopJanitor(s *memoryStorage) {
	s.janitor.stop <- true
}

func runJanitor(s *memoryStorage, ci time.Duration) {
	j := &janitor{
		Interval: ci,
	}
	s.janitor = j
	go j.Run(s)
}