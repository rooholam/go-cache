package cache

import (
	"fmt"
	"strconv"
	"strings"
	"bytes"
	"time"

	redis "gopkg.in/redis.v4"
	log "github.com/Sirupsen/logrus"
	lock "github.com/bsm/redis-lock"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)

type redisStorage struct {
	redisClient *redis.Client
	marshaller  *runtime.JSONPb
	lock        *lock.Lock
}

func (s *redisStorage) Get(key string) (Item, bool) {
	res, err := s.redisClient.Get(key).Result()
	if err != nil {
		return Item{}, false
	}

	return s.UnMarshal(res, nil), true
}

func (s *redisStorage) GetObject(key string, o interface{}) (Item, bool) {
	res, err := s.redisClient.Get(key).Result()
	if err != nil {
		return Item{}, false
	}

	return s.UnMarshal(res, o), true
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

func (s *redisStorage) Marshal(m Item) string {
	res, err := s.marshaller.Marshal(m.Object)
	if err != nil {
		log.Errorf("error marshaling : %s", err)
	}
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%d|%d|", m.Expiration, m.RefreshDeadline))
	buf.Write(res)
	out := buf.String()
	return out
}

func (s *redisStorage) UnMarshal(m string, o interface{}) Item {
	var item Item
	res := strings.SplitN(m, "|", 3)
	item.Expiration, _ = strconv.ParseInt(res[0], 10, 64)
	item.RefreshDeadline, _ = strconv.ParseInt(res[1], 10, 64)

	err := s.marshaller.NewDecoder(strings.NewReader(res[2])).Decode(o)
	if err != nil {
		log.Errorf("error unmarshaling : %s", err)
	}
	item.Object = o
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
		marshaller:&runtime.JSONPb{OrigName: true},
	}

	return &red
}
