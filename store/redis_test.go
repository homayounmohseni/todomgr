package store

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/alicebob/miniredis/v2"
)

func TestRedisCacheRoundTrip(t *testing.T) {
	mr := miniredis.RunT(t)
	c := NewRedisCache(redis.NewClient(&redis.Options{Addr: mr.Addr()}))
	ctx := context.Background()

	if _, err := c.Get(ctx, "task:1"); err == nil {
		t.Fatal("want miss")
	}
	if err := c.Set(ctx, "task:1", `{"id":1}`, time.Minute); err != nil {
		t.Fatal(err)
	}
	got, err := c.Get(ctx, "task:1")
	if err != nil || got != `{"id":1}` {
		t.Fatalf("got %q err %v", got, err)
	}
	if err := c.Del(ctx, "task:1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, "task:1"); err == nil {
		t.Fatal("want miss after del")
	}
}
