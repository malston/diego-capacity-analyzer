package cache

import (
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	c := New(1 * time.Second)

	c.Set("key1", "value1")

	val, found := c.Get("key1")
	if !found {
		t.Error("Expected to find key1")
	}
	if val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}
}

func TestCache_Expiration(t *testing.T) {
	c := New(100 * time.Millisecond)

	c.Set("key1", "value1")

	// Should exist immediately
	_, found := c.Get("key1")
	if !found {
		t.Error("Expected to find key1 immediately")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	_, found = c.Get("key1")
	if found {
		t.Error("Expected key1 to be expired")
	}
}

func TestCache_Clear(t *testing.T) {
	c := New(1 * time.Second)

	c.Set("key1", "value1")
	c.Clear("key1")

	_, found := c.Get("key1")
	if found {
		t.Error("Expected key1 to be cleared")
	}
}
