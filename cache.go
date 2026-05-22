package ironweft

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"sync"
	"time"
)

type cacheEntry struct {
	value     *AuthorizeResponse
	expiresAt time.Time
}

// AuthCache is a thread-safe in-process cache for allow decisions.
// TTL is bound to the credential's JWT expiry. Only allow decisions are cached.
type AuthCache struct {
	mu    sync.RWMutex
	store map[string]cacheEntry
}

func newAuthCache() *AuthCache {
	return &AuthCache{store: make(map[string]cacheEntry)}
}

func jwtExp(credential string) (time.Time, bool) {
	parts := strings.SplitN(credential, ".", 3)
	if len(parts) < 2 {
		return time.Time{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil || claims.Exp == 0 {
		return time.Time{}, false
	}
	return time.Unix(claims.Exp, 0), true
}

func buildCacheKey(credential, action, resource string, parameters map[string]interface{}) string {
	// encoding/json marshals map keys in sorted order.
	b, _ := json.Marshal(parameters)
	return credential + "\x00" + action + "\x00" + resource + "\x00" + string(b)
}

// Get returns a cached allow response, or nil/false on miss or expiry.
func (c *AuthCache) Get(credential, action, resource string, parameters map[string]interface{}) (*AuthorizeResponse, bool) {
	key := buildCacheKey(credential, action, resource, parameters)
	c.mu.RLock()
	entry, ok := c.store[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.store, key)
		c.mu.Unlock()
		return nil, false
	}
	cp := *entry.value
	cp.Cached = true
	return &cp, true
}

// Set caches a response. Only stores if the credential's JWT is not yet expired.
func (c *AuthCache) Set(credential, action, resource string, parameters map[string]interface{}, resp *AuthorizeResponse) {
	exp, ok := jwtExp(credential)
	if !ok || time.Now().After(exp) {
		return
	}
	key := buildCacheKey(credential, action, resource, parameters)
	c.mu.Lock()
	c.store[key] = cacheEntry{value: resp, expiresAt: exp}
	c.mu.Unlock()
}

// InvalidateCredential evicts all cached entries for the given credential.
func (c *AuthCache) InvalidateCredential(credential string) {
	prefix := credential + "\x00"
	c.mu.Lock()
	for k := range c.store {
		if strings.HasPrefix(k, prefix) {
			delete(c.store, k)
		}
	}
	c.mu.Unlock()
}

// Clear evicts all cached entries.
func (c *AuthCache) Clear() {
	c.mu.Lock()
	c.store = make(map[string]cacheEntry)
	c.mu.Unlock()
}
