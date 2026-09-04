package models

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTTLAreaTreeCacheBasics(t *testing.T) {
	c := newTTLAreaTreeCache(time.Minute)

	_, ok := c.get()
	assert.False(t, ok, "空缓存应未命中")

	tree := AreaModelList{{Value: "11", Label: "北京市"}}
	c.set(tree)
	got, ok := c.get()
	assert.True(t, ok, "写入后应命中")
	assert.Equal(t, tree, got)

	c.invalidate()
	_, ok = c.get()
	assert.False(t, ok, "失效后应未命中")
}

func TestTTLAreaTreeCacheExpired(t *testing.T) {
	c := newTTLAreaTreeCache(time.Millisecond)
	c.set(AreaModelList{{Value: "11"}})
	time.Sleep(5 * time.Millisecond)
	_, ok := c.get()
	assert.False(t, ok, "过期后应未命中")
}

func TestTTLAreaTreeCacheConcurrentAccess(t *testing.T) {
	c := newTTLAreaTreeCache(time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); c.set(AreaModelList{{Value: "11"}}) }()
		go func() { defer wg.Done(); _, _ = c.get() }()
		go func() { defer wg.Done(); c.invalidate() }()
	}
	wg.Wait()
}
