package cache

import (
	"fmt"
	"maps"
	"net"
	"slices"
	"sync"
	"time"
)

const (
	// ConfigResolver is a ResolverFunc.
	ConfigResolver = ConfigKey("Resolver")
	// ConfigRefreshShuffle is a bool.
	// True will "shuffle" cache items before the Refresh begins.
	ConfigRefreshShuffle = ConfigKey("RefreshShuffle")
	// ConfigRefreshSleepTime is a time.Duration.
	// If > 0 then a Refresh pass, if any, will wait that Duration between item lookups.
	// 0 disables the delay.
	ConfigRefreshSleepTime = ConfigKey("RefreshSleepTime")
)

// Simple is a mutex-controlled map-based ResolverCache.
type Simple struct {
	lock  sync.RWMutex
	cache map[string][]net.IP
	done  chan struct{}

	resolver         ResolverFunc
	refreshShuffle   bool
	refreshSleepTime time.Duration
	refreshType      RefreshType
	refresh          RefreshFunc
	refreshBatchSize int
}

// NewSimple instantiates a Simple cache.
// Valid ConfigOptions are: Resolver, RefreshShuffle, RefreshSleepTime.
// Required are: none.
// Defaults are: Resolver(DefaultResolver), RefreshShuffle(true), RefreshSleepTime(1s)
func NewSimple(options ...ConfigOption) (*Simple, error) {
	s := Simple{
		cache:            make(map[string][]net.IP, 64),
		done:             make(chan struct{}),
		refreshShuffle:   true,
		refreshSleepTime: 1 * time.Second,
		resolver:         DefaultResolver,
		refresh:          LinearRefresh,
		refreshType:      RefreshLinear,
		refreshBatchSize: 15,
	}

	// Apply options
	var e error
	for _, o := range options {
		e = s.config(o)
		if e != nil {
			return nil, e
		}
	}

	return &s, nil
}

// config is an internal validator and applier for ConfigOptions
func (r *Simple) config(opt ConfigOption) error {
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
	case ConfigRefreshType:
		if v, ok := opt.Value.(RefreshType); ok {
			r.refreshType = v
			switch v {
			case RefreshOff:
				r.refresh = NoRefresh
			case RefreshLinear:
				r.refresh = LinearRefresh
			case RefreshBatch:
				r.refresh = BatchRefresh

			}
		} else {
			return opt.Key.Error()
		}
	case ConfigRefreshBatchSize:
		if v, ok := opt.Value.(int); ok {
			r.refreshBatchSize = v
		} else {
			return opt.Key.Error()
		}
	default:
		return ErrorConfigKeyUnsupported
	}
	return nil
}

// Fetch retrieves a collection from the cache,
// or performs a live lookup and adds it to the cache.
func (r *Simple) Fetch(address string) ([]net.IP, error) {
	r.lock.RLock()
	ips, exists := r.cache[address]
	r.lock.RUnlock()
	if exists {
		return ips, nil
	}

	return r.Lookup(address)
}

// Lookup returns a collection of IPs from a live lookup, and updates the cache.
// Most callers should use one of the Fetch functions.
func (r *Simple) Lookup(address string) ([]net.IP, error) {
	ips, err := r.resolver(address)
	if err != nil {
		return nil, err
	}

	r.lock.Lock()
	r.cache[address] = ips
	r.lock.Unlock()
	return ips, nil
}

// Purge removes all entries from the cache.
func (r *Simple) Purge() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.cache = make(map[string][]net.IP, 64)
}

// Refresh will crawl the cache and update their entries.
// A timeout of 0 must mean no timeout.
// RefreshSleepTime is checked for per-lookup intervals.
// RefreshShuffle is checked.
func (r *Simple) Refresh(timeout time.Duration) {
	var err error

	if r.refreshType != RefreshBatch {
		_, err = r.refresh(r, r.Lookup,
			NewConfigOption(ConfigRefreshShuffle, r.refreshShuffle),
			NewConfigOption(ConfigRefreshSleepTime, r.refreshSleepTime),
			NewConfigOption(ConfigRefreshTimeout, timeout),
		)
	} else {
		// batch
		_, err = r.refresh(r, r.Lookup,
			NewConfigOption(ConfigRefreshShuffle, r.refreshShuffle),
			NewConfigOption(ConfigRefreshSleepTime, r.refreshSleepTime),
			NewConfigOption(ConfigRefreshTimeout, timeout),
			NewConfigOption(ConfigRefreshBatchSize, r.refreshBatchSize),
		)
	}

	if err != nil {
		panic(fmt.Errorf("error during RefreshFunc: %w", err))
	}
}

// Close will signal an in-progress Refresh, if any, to exit.
func (r *Simple) Close() error {
	close(r.done)
	return nil
}

// Add will upsert a collection into the cache.
func (r *Simple) Add(address string, ips []net.IP) {
	r.lock.Lock()
	r.cache[address] = ips
	r.lock.Unlock()
}

// Remove will remove a collection from the cache, if it exists.
func (r *Simple) Remove(address string) {
	r.lock.Lock()
	delete(r.cache, address)
	r.lock.Unlock()
}

// Get will return a collection from the cache, also bool if
// a collection was retrieved.
func (r *Simple) Get(address string) ([]net.IP, bool) {
	r.lock.RLock()
	v, ok := r.cache[address]
	r.lock.RUnlock()

	return v, ok
}

// Len will return the number of items in the cache.
func (r *Simple) Len() int {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return len(r.cache)
}

// Contains returns true if a value is in the cache.
func (r *Simple) Contains(address string) bool {
	r.lock.RLock()
	_, ok := r.cache[address]
	r.lock.RUnlock()

	return ok
}

// Keys returns a sorted slice of the cache keys
func (r *Simple) Keys() []string {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return slices.Sorted(maps.Keys(r.cache))
}
