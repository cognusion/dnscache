// Package dnscache caches DNS lookups.
// The package itself requires no non-standard modules, however
// a separate testing suite is used.
//
// Based on https://github.com/viki-org/dnscache with modern Go
// facilities, no intrinsic goro leak, more flexibility, and more.
package dnscache

import (
	"context"
	"maps"
	"math/rand/v2"
	"net"
	"slices"
	"sync"
	"time"
)

var (
	// RefreshSleepTime is the delay between Refresh (and auto-refresh)
	// lookups, to keep the resolver threads from piling up.
	RefreshSleepTime = 1 * time.Second

	// RefreshShuffle is used to control whether the addresses are shuffled during Refresh,
	// to avoid long-tail misses on sufficiently large caches.
	RefreshShuffle = true
)

// Resolver is a goro-safe caching DNS resolver.
type Resolver struct {
	lock  sync.RWMutex
	cache map[string][]net.IP
	done  chan struct{}
}

// New returns a properly instantiated Resolver.
// If the refreshRate is non-zero, a goro will refresh
// all of the entries after that Duration.
func New(refreshRate time.Duration) *Resolver {
	return NewWithRefreshTimeout(refreshRate, 0)
}

// NewWithRefreshTimeout returns a properly instantiated Resolver.
// If the refreshRate is non-zero, a goro will refresh
// all of the entries after that Duration.
// If refreshTimeout is non-zero, each auto-refresh iteraction will timeout after
// the specified duration (expressed as a deadline).
func NewWithRefreshTimeout(refreshRate, refreshTimeout time.Duration) *Resolver {
	resolver := &Resolver{
		cache: make(map[string][]net.IP, 64),
		done:  make(chan struct{}),
	}
	if refreshRate > 0 {
		go resolver.autoRefreshTimeout(refreshRate, refreshTimeout)
	}
	return resolver
}

// Close signals the auto-refresh goro, if any, to quit.
// This is safe to call once, in any thread, regardless of whether or not auto-refresh is used.
func (r *Resolver) Close() error {
	close(r.done)
	return nil
}

// Fetch returns a collection of IPs from cache, or a live lookup if not.
func (r *Resolver) Fetch(address string) ([]net.IP, error) {
	r.lock.RLock()
	ips, exists := r.cache[address]
	r.lock.RUnlock()
	if exists {
		return ips, nil
	}

	return r.Lookup(address)
}

// FetchOne returns a single IP from cache, or a live lookup if not.
func (r *Resolver) FetchOne(address string) (net.IP, error) {
	ips, err := r.Fetch(address)
	if err != nil || len(ips) == 0 {
		return nil, err
	}
	return ips[0], nil
}

// FetchOneString returns a single IP -as a string- from cache, or a live lookup if not.
func (r *Resolver) FetchOneString(address string) (string, error) {
	ip, err := r.FetchOne(address)
	if err != nil || ip == nil {
		return "", err
	}
	return ip.String(), nil
}

// Refresh will iterate over cache items, and performing a live lookup one every RefreshSleepTime.
func (r *Resolver) Refresh() {
	r.RefreshTimeout(0)
}

// RefreshTimeout will iterate over cache items, and performing a live lookup one every RefreshSleepTime,
// until completed or the stated timeout expires.
func (r *Resolver) RefreshTimeout(timeout time.Duration) {

	r.lock.RLock()
	addressesIter := maps.Keys(r.cache)
	r.lock.RUnlock()

	// Create the iterator funcs we'll use. This also
	addresses := slices.Sorted(addressesIter)

	if len(addresses) == 0 {
		// empty cache
		return
	}

	if RefreshShuffle {
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
		case <-time.After(RefreshSleepTime):
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

// Lookup returns a collection of IPs from a live lookup, and updates the cache.
// Most callers should use one of the Fetch functions.
func (r *Resolver) Lookup(address string) ([]net.IP, error) {
	ips, err := net.LookupIP(address)
	if err != nil {
		return nil, err
	}

	r.lock.Lock()
	r.cache[address] = ips
	r.lock.Unlock()
	return ips, nil
}

// autoRefresh is an internal loop to Refresh every declared interval.
// The loop terminates if Close is called.
// The specified timeout is passed on to each Refresh iteration, or 0 for
// no timeout.
func (r *Resolver) autoRefreshTimeout(rate, timeout time.Duration) {
	for {
		select {
		case <-time.After(rate):
			r.RefreshTimeout(timeout)
		case <-r.done:
			return
		}
	}
}
