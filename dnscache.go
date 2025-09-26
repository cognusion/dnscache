// Package dnscache caches DNS lookups.
// The package itself requires no non-standard modules, however
// a separate testing suite is used.
//
// Based on https://github.com/viki-org/dnscache with modern Go
// facilities, no intrinsic goro leak, more flexibility, and more.
package dnscache

import (
	"fmt"
	"net"
	"time"

	"github.com/cognusion/dnscache/cache"
)

var (
	// RefreshSleepTime is the delay between Refresh (and auto-refresh)
	// lookups, to keep the resolver threads from piling up.
	// Changes after a Resolver is instantiated are ignored.
	RefreshSleepTime = 1 * time.Second

	// RefreshShuffle is used to control whether the addresses are shuffled during Refresh,
	// to avoid long-tail misses on sufficiently large caches.
	// Changes after a Resolver is instantiated are ignored.
	RefreshShuffle = true
)

// Resolver is a goro-safe caching DNS resolver.
type Resolver struct {
	cache  ResolverCache
	config *ResolverConfig
	done   chan struct{}
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
	config := &ResolverConfig{
		AutoRefreshInterval: refreshRate,
		AutoRefreshTimeout:  refreshTimeout,
	}

	var err error
	config.Cache, err = cache.NewSimple(
		cache.NewConfigOption(cache.ConfigRefreshSleepTime, RefreshSleepTime),
		cache.NewConfigOption(cache.ConfigRefreshShuffle, RefreshShuffle),
	)
	if err != nil {
		panic(fmt.Errorf("impossible error occurred creating a cache.Simple: %w", err))
	}
	return NewFromConfig(config)
}

// NewFromConfig returns a properly instantiated resolver, using the provided Cache
// and the provided AutoRefresh* values.
// NOTE: If using an LRU-style cache, setting the AutoRefreshInterval as large as
// feasible is advised, to keep the cache calculus correct.
func NewFromConfig(config *ResolverConfig) *Resolver {
	if config.Cache == nil {
		// cache wasn't specified. Why is this constructor called?!
		c, _ := cache.NewSimple() // defaults, no error trap needed
		config.Cache = c
	}

	resolver := &Resolver{
		cache:  config.Cache,
		config: config,
		done:   make(chan struct{}),
	}

	if config.AutoRefreshInterval > 0 {
		go resolver.autoRefreshTimeout(config.AutoRefreshInterval, config.AutoRefreshTimeout)
	}

	return resolver
}

// Close signals the auto-refresh goro, if any, to quit.
// This is safe to call once, in any thread, regardless of whether or not auto-refresh is used.
func (r *Resolver) Close() error {
	close(r.done)
	return r.cache.Close()
}

// Fetch returns a collection of IPs from cache, or a live lookup if not.
func (r *Resolver) Fetch(address string) ([]net.IP, error) {
	return r.cache.Fetch(address)
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
	r.cache.Refresh(0)
}

// RefreshTimeout will iterate over cache items, and performing a live lookup one every RefreshSleepTime,
// until completed or the stated timeout expires.
func (r *Resolver) RefreshTimeout(timeout time.Duration) {
	r.cache.Refresh(timeout)
}

// Lookup returns a collection of IPs from a live lookup, and updates the cache.
// Most callers should use one of the Fetch functions.
func (r *Resolver) Lookup(address string) ([]net.IP, error) {
	return r.cache.Lookup(address)
}

// Purge will remove all entries. To comply with ResolverCache.
func (r *Resolver) Purge() {
	r.cache.Purge()
}

// autoRefresh is an internal loop to Refresh every declared interval.
// The loop terminates if Close is called.
// The specified timeout is passed on to each Refresh iteration, or 0 for
// no timeout.
func (r *Resolver) autoRefreshTimeout(rate, timeout time.Duration) {
	for {
		select {
		case <-time.After(rate):
			r.cache.Refresh(timeout)
		case <-r.done:
			return
		}
	}
}
