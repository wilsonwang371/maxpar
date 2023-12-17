package cache

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	// Test the cache
	expiration := time.Second * 1
	cleanupInterval := time.Second * 2

	c := New(expiration, cleanupInterval)
	c.Set("foo", "bar", DefaultExpiration)
	if x, found := c.Get("foo"); !found || x != "bar" {
		t.Error("cache set foo failed")
	}
	if _, found := c.Get("bar"); found {
		t.Error("cache set bar failed")
	}
	c.Set("foo", "bar", DefaultExpiration)
	c.Delete("foo")
	if _, found := c.Get("foo"); found {
		t.Error("cache delete foo failed")
	}
}

func TestCacheTimeout(t *testing.T) {
	// Test the cache
	expiration := time.Second * 1
	cleanupInterval := time.Second * 2

	c := New(expiration, cleanupInterval)
	c.Set("foo", "bar", DefaultExpiration)
	time.Sleep(expiration + time.Second)
	if _, found := c.Get("foo"); found {
		t.Error("cache timeout failed")
	}
}
