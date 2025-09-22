package cache

import (
	"context"
	"maps"
	"math/rand/v2"
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
	r.lock.RLock()
	addressesIter := maps.Keys(r.cache)
	r.lock.RUnlock()

	// Create the iterator funcs we'll use. This also
	addresses := slices.Sorted(addressesIter)

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
			r.Lookup(addresses[i])
		case <-r.done:
			// actively cancelled
			return
		case <-ctx.Done():
			// took too long, deadline exceeded.
			return
		}
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
