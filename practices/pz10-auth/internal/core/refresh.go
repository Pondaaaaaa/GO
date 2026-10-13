package core

import (
	"sync"
	"time"
)

type RefreshBlacklist struct {
	mu      sync.Mutex
	revoked map[string]time.Time
}

func NewRefreshBlacklist() *RefreshBlacklist {
	return &RefreshBlacklist{revoked: make(map[string]time.Time)}
}
func (b *RefreshBlacklist) RevokeOnce(id string, expires time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	for key, exp := range b.revoked {
		if !exp.After(now) {
			delete(b.revoked, key)
		}
	}
	if _, exists := b.revoked[id]; exists {
		return false
	}
	b.revoked[id] = expires
	return true
}
