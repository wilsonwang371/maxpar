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
	c.Set("foo", "bar", NoSpecifiedExpiration)
	if x, found := c.Get("foo"); !found || x != "bar" {
		t.Error("cache set foo failed")
	}
	if _, found := c.Get("bar"); found {
		t.Error("cache set bar failed")
	}
	c.Set("foo", "bar", NoSpecifiedExpiration)
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
	c.Set("foo", "bar", NoSpecifiedExpiration)
	time.Sleep(expiration + time.Second)
	if _, found := c.Get("foo"); found {
		t.Error("cache timeout failed")
	}
}

func TestCacheCount(t *testing.T) {
	// Test the cache
	expiration := time.Second * 1
	cleanupInterval := time.Second * 2

	c := New(expiration, cleanupInterval)
	c.Set("foo", "bar", NoSpecifiedExpiration)
	c.Set("foo2", "bar2", NoSpecifiedExpiration)
	if c.Count() != 2 {
		t.Error("cache count failed")
	}

	c.Delete("foo")
	if c.Count() != 1 {
		t.Error("cache count failed")
	}

	c.Delete("foo2")
	if c.Count() != 0 {
		t.Error("cache count failed")
	}

	c.Set("foo", "bar", NoSpecifiedExpiration)
	c.Set("foo2", "bar2", NoSpecifiedExpiration)
	// test after timeout
	time.Sleep(expiration + time.Second)
	if c.Count() != 0 {
		t.Error("cache count failed")
	}
}
