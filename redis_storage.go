package cache

import (
	"encoding/json"
	redis "gopkg.in/redis.v4"
	log "github.com/Sirupsen/logrus"
	lock "github.com/bsm/redis-lock"
	"time"
)

type redisStorage struct {
	redisClient *redis.Client
	lock        *lock.Lock
}

func (s *redisStorage) Get(key string) (Item, bool) {
	res, err := s.redisClient.Get(key).Result()
	if err != nil {
		return Item{}, false
	}
	return s.UnMarshal(res), true

}

func (s *redisStorage) Set(key string, item Item) {
	s.redisClient.Set(key, s.Marshal(item), time.Unix(0, item.Expiration).Sub(time.Now()))
}

func (s *redisStorage) Del(key string) {
	s.redisClient.Del(key)
}

func (s *redisStorage) Flush() {
	s.redisClient.FlushDb()
}

func (s *redisStorage) Lock() {
	s.lock.Lock()
}

func (s *redisStorage) Unlock() {
	s.lock.Unlock()
}

func (s *redisStorage) RLock() {
	// nothing to do
}

func (s *redisStorage) RUnlock() {
	// nothing to do
}

func (s *redisStorage) Type() int {
	return STORAGE_TYPE_REDIS
}

func (s *redisStorage) Marshal(item Item) string {
	b, err := json.Marshal(item)
	if err != nil {
		log.Errorf("error marshalling %s", err)
	}
	return string(b)
}

func (s *redisStorage) UnMarshal(str string) Item {
	var item Item
	err := json.Unmarshal([]byte(str), &item)
	if err != nil {
		log.Errorf("error unmarshalling %s", err)
	}
	return item
}

func RedisStorage(addr string, pass string, db int) *redisStorage {
	opts := &redis.Options{
		Addr:     addr,
		Password: pass,
		DB:       db,
	}

	client := redis.NewClient(opts)
	_, err := client.Ping().Result()
	if err != nil {
		log.Errorf("failed to initialize DataCollector redisClient: %s", err)
	}

	lock, err := lock.ObtainLock(client, "go_cache_lock", nil)
	if err != nil {
		log.Errorf("ERROR: %s\n", err.Error())
	} else if lock == nil {
		log.Errorf("ERROR: could not obtain lock")
	}

	red := redisStorage{
		redisClient:client,
		lock:lock,
	}
	return &red
}
