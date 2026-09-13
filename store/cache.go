package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/homayounmohseni/todomgr/model"
	"github.com/homayounmohseni/todomgr/observability"

	"github.com/redis/go-redis/v9"
)

const (
	taskKeyPrefix = "task:"
	defaultTTL    = time.Minute
)

type TaskCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
}

type redisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) TaskCache {
	return &redisCache{client: client}
}

func (c *redisCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *redisCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *redisCache) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

type CachedTaskStore struct {
	store TaskStore
	cache TaskCache
	ttl   time.Duration
}

func NewCachedTaskStore(s TaskStore, c TaskCache, ttl time.Duration) *CachedTaskStore {
	if ttl <= 0 {
		ttl = defaultTTL
	}
	return &CachedTaskStore{store: s, cache: c, ttl: ttl}
}

func taskKey(id int64) string {
	return fmt.Sprintf("%s%d", taskKeyPrefix, id)
}

func (s *CachedTaskStore) List(ctx context.Context, f ListFilter) ([]model.Task, error) {
	return s.store.List(ctx, f)
}

func (s *CachedTaskStore) Get(ctx context.Context, id int64) (model.Task, error) {
	key := taskKey(id)
	if hit, err := s.cache.Get(ctx, key); err == nil {
		var t model.Task
		if err := json.Unmarshal([]byte(hit), &t); err == nil {
			observability.CacheHit()
			return t, nil
		}
	}
	observability.CacheMiss()
	t, err := s.store.Get(ctx, id)
	if err != nil {
		return model.Task{}, err
	}
	if raw, err := json.Marshal(t); err == nil {
		_ = s.cache.Set(ctx, key, string(raw), s.ttl)
	}
	return t, nil
}

func (s *CachedTaskStore) Create(ctx context.Context, title, assignee string) (model.Task, error) {
	return s.store.Create(ctx, title, assignee)
}

func (s *CachedTaskStore) Update(ctx context.Context, id int64, title string, status bool, assignee string) (model.Task, error) {
	t, err := s.store.Update(ctx, id, title, status, assignee)
	if err != nil {
		return model.Task{}, err
	}
	_ = s.cache.Del(ctx, taskKey(id))
	return t, nil
}

func (s *CachedTaskStore) Delete(ctx context.Context, id int64) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.cache.Del(ctx, taskKey(id))
	return nil
}
