package cache

import (
	"time"
	"bytes"
	"encoding/gob"
	"fmt"
	"encoding/base64"

	redis "gopkg.in/redis.v4"
	log "github.com/Sirupsen/logrus"
	lock "github.com/bsm/redis-lock"
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
	return s.FromGOB64(res), true

}

func (s *redisStorage) Set(key string, item Item) {
	s.redisClient.Set(key, s.ToGOB64(item), time.Unix(0, item.Expiration).Sub(time.Now()))
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

func (s *redisStorage) ToGOB64(m Item) string {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(m)
	if err != nil {
		fmt.Println(`failed gob Encode`, err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

func (s *redisStorage) FromGOB64(str string) Item {
	m := Item{}
	by, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		fmt.Println(`failed base64 Decode`, err); }
	b := bytes.Buffer{}
	b.Write(by)
	d := gob.NewDecoder(&b)
	err = d.Decode(&m)
	if err != nil {
		fmt.Println(`failed gob Decode`, err); }
	return m
}

func RedisStorage(addr string, pass string, db int, objectTypes ...interface{}) *redisStorage {
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
	gob.Register(Item{})
	for _, obj := range objectTypes {
		gob.Register(obj)
	}
	return &red
}
