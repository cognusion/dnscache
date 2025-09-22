package cache

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/hashicorp/golang-lru/v2/expirable"
)

const (
	// ConfigSize is an int.
	// Values > 0 represent the number of items allowed in the cache.
	// 0 is unlimited.
	ConfigSize = ConfigKey("CacheSize")
	// ConfigItemTTL is a time.Duration.
	// Values will control the life of unaccessed items in the cache.
	ConfigItemTTL = ConfigKey("ItemTTL")
	// ConfigAllowRefresh is a bool.
	// True allows the cache to perform Refresh operations.
	// False requires the cache to silently decline Refresh operations.
	ConfigAllowRefresh = ConfigKey("AllowRefresh")
)

// hashiLRU is an abstraction to let us reuse LRU, but support multiple LRU types via
// different constructors.
type hashiLRU interface {
	Add(key string, value []net.IP)
	Contains(key string) bool
	Get(key string) (value []net.IP, ok bool)
	Remove(key string)
	Keys() []string
	Len() int
	Purge()
}

// I don't want to talk about it
type expirableWrapper struct {
	*expirable.LRU[string, []net.IP]
}

func (e *expirableWrapper) Add(key string, value []net.IP) {
	e.LRU.Add(key, value) // ignores the bool returned.
}
func (e *expirableWrapper) Remove(key string) {
	e.LRU.Remove(key) //ignores the bool returned.
}

// LRU is a "least recently used" cache of fixed size, that evicts items
// when necessary to free space for more. If ItemTTL is specified, then
// the cache will automatically evict items that are unaccessed beyond that point.
type LRU struct {
	cache hashiLRU

	allowRefresh     bool
	resolver         ResolverFunc
	refreshShuffle   bool
	refreshSleepTime time.Duration
}

// NewLRU instantiates an LRU cache.
// If ItemTTL is specified, an expirable cache is created, otherwise a twoqueue cache is used.
// Valid ConfigOptions are: Resolver, RefreshShuffle, RefreshSleepTime, AllowRefresh, ItemTTL, Size.
// Required are: Size.
// Defaults are: Resolver(DefaultResolver), RefreshShuffle(true), RefreshSleepTime(1s), AllowRefresh(true).
func NewLRU(options ...ConfigOption) (*LRU, error) {
	var cacheSize int
	if v, ok := ConfigSize.IsIn(options); !ok {
		return nil, fmt.Errorf("option %s is required", ConfigSize)
	} else if cacheSize, ok = v.(int); !ok {
		return nil, ConfigSize.Error()
	}

	var (
		cache hashiLRU
		err   error
		ttl   time.Duration
	)

	// Requirements
	if v, ok := ConfigItemTTL.IsIn(options); ok {
		// We want an expirable cache
		if ttl, ok = v.(time.Duration); !ok {
			return nil, ConfigItemTTL.Error()
		}
		cache = &expirableWrapper{expirable.NewLRU[string, []net.IP](cacheSize, nil, ttl)}
	} else {
		// We do not want an expirable cache
		cache, err = lru.New2Q[string, []net.IP](cacheSize)
	}
	if err != nil {
		return nil, fmt.Errorf("error instantiating lru: %w", err)
	}

	// Set defaults
	l := LRU{
		cache:            cache,
		refreshShuffle:   true,
		refreshSleepTime: 1 * time.Second,
		resolver:         DefaultResolver,
		allowRefresh:     true,
	}

	// Apply options
	var e error
	for _, o := range options {
		e = l.config(o)
		if e != nil {
			return nil, e
		}
	}

	return &l, nil
}

// config is an internal validator and applier for ConfigOptions
func (r *LRU) config(opt ConfigOption) error {
	switch opt.Key {
	case ConfigResolver:
		if v, ok := opt.Value.(ResolverFunc); ok {
			r.resolver = v
		} else {
			return opt.Key.Error()
		}
	case ConfigRefreshShuffle:
		if v, ok := opt.Value.(bool); ok {
			r.refreshShuffle = v
		} else {
			return opt.Key.Error()
		}
	case ConfigRefreshSleepTime:
		if v, ok := opt.Value.(time.Duration); ok {
			r.refreshSleepTime = v
		} else {
			return opt.Key.Error()
		}
	case ConfigAllowRefresh:
		if v, ok := opt.Value.(bool); ok {
			r.allowRefresh = v
		} else {
			return opt.Key.Error()
		}
	case ConfigItemTTL:
		// supported in constructor, but not changeable. Type test for funsies.
		if _, ok := opt.Value.(time.Duration); !ok {
			return opt.Key.Error()
		}
	case ConfigSize:
		// supported in constructor, but not changeable. Type test for funsies.
		if _, ok := opt.Value.(int); !ok {
			return opt.Key.Error()
		}
	default:
		return ErrorConfigKeyUnsupported
	}
	return nil
}

// Fetch retrieves a collection from the cache,
// or performs a live lookup and adds it to the cache.
func (r *LRU) Fetch(address string) ([]net.IP, error) {
	ips, exists := r.cache.Get(address)
	if exists {
		return ips, nil
	}

	return r.Lookup(address)
}

// Lookup performs a live lookup,
// and adds the results to the cache.
func (r *LRU) Lookup(address string) ([]net.IP, error) {
	ips, err := r.resolver(address)
	if err != nil {
		return nil, err
	}

	r.cache.Add(address, ips)
	return ips, nil
}

// Purge removes all entries from the cache.
func (r *LRU) Purge() {
	r.cache.Purge()
}

// Refresh will crawl the keys and update the cache with new values.
func (r *LRU) Refresh(timeout time.Duration) {
	if !r.allowRefresh {
		// nope
		return
	}

	// Get the keys
	addresses := r.cache.Keys()

	if len(addresses) == 0 {
		// empty cache
		return
	}

	if r.refreshShuffle {
		rand.Shuffle(len(addresses), func(i, j int) {
			addresses[i], addresses[j] = addresses[j], addresses[i]
		})
	}

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	if timeout == 0 {
		// No deadline
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		// Deadline
		ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(timeout))
	}
	defer cancel() // because yes

	// first lookup is out of loop, so we don't wait
	r.Lookup(addresses[0])

	// offset i to account for the previous lookup
	for i := 1; i < len(addresses); i++ {
		select {
		case <-time.After(r.refreshSleepTime):
			// this loop is here because it is highly possible that one or more of the
			// previously-existing addresses no longer is in the cache, due to
			// pressure or TTL evictions. So we peek into the cache to see if an
			// address still exists, until one finally does, then we break out and
			// outer-loop again.
		STALE:
			for {
				if i >= len(addresses) {
					// that's all folks
					return
				}
				if r.cache.Contains(addresses[i]) {
					r.Lookup(addresses[i])
					break STALE
				}
				i++
			}
		case <-ctx.Done():
			// took too long, deadline exceeded.
			return
		}
	}
}

// Close is a noop. Satisfies ResolverCache
func (r *LRU) Close() error {
	return nil
}

// Add will upsert a collection into the cache.
func (r *LRU) Add(key string, value []net.IP) {
	r.cache.Add(key, value)
}

// Remove will remove a collection from the cache, if it exists.
func (r *LRU) Remove(key string) {
	r.cache.Remove(key)
}

// Get will return a collection from the cache, also bool if
// a collection was retrieved.
func (r *LRU) Get(key string) ([]net.IP, bool) {
	return r.cache.Get(key)
}

// Len will return the number of items in the cache.
func (r *LRU) Len() int {
	return r.cache.Len()
}
