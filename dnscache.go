// Package dnscache caches DNS lookups.
// The package itself requires no non-standard modules, however
// a separate testing suite is used.
//
// Based on https://github.com/viki-org/dnscache with modern Go
// facilities, no intrinsic goro leak, more flexibility, and more.
package dnscache

import (
	"maps"
	"net"
	"sync"
	"time"
)

// RefreshSleepTime is the delay between Refresh (and auto-refresh)
// lookups, to keep the resolver threads from piling up.
var RefreshSleepTime = 1 * time.Second

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
	resolver := &Resolver{
		cache: make(map[string][]net.IP, 64),
		done:  make(chan struct{}),
	}
	if refreshRate > 0 {
		go resolver.autoRefresh(refreshRate)
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
	r.lock.RLock()
	addresses := maps.Keys(r.cache)
	r.lock.RUnlock()

	for address := range addresses {
		r.Lookup(address)
		time.Sleep(RefreshSleepTime)
	}
}

// Lookup returns a collection of IPs from a live lookup, and updates the cache.
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
func (r *Resolver) autoRefresh(rate time.Duration) {
	for {
		select {
		case <-time.After(rate):
			r.Refresh()
		case <-r.done:
			return
		}
	}
}
