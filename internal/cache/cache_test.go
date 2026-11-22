package cache

import (
	"bytes"
	"testing"
	"time"
)

func TestCacheAddGetAndReap(t *testing.T) {
	c := NewCache(40 * time.Millisecond)

	c.Add("goblin", []byte(`{"name":"Goblin"}`))
	got, ok := c.Get("goblin")
	if !ok {
		t.Fatal("expected cache hit for goblin")
	}
	if !bytes.Equal(got, []byte(`{"name":"Goblin"}`)) {
		t.Fatalf("unexpected cached value: %s", got)
	}

	time.Sleep(80 * time.Millisecond)

	if _, ok := c.Get("goblin"); ok {
		t.Fatal("expected cache entry to expire after TTL")
	}
}
