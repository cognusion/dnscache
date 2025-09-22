package dnscache

import (
	"net"
	"time"
)

// ResolverCache is an interface to define different caches for Resolver.
// All functions defined here must be goro-safe.
type ResolverCache interface {
	// Fetch retrieves a collection from the cache,
	// or performs a live lookup and adds it to the cache.
	Fetch(string) ([]net.IP, error)
	// Lookup performs a live lookup,
	// and adds the results to the cache.
	Lookup(address string) ([]net.IP, error)
	// Purge removes all entries from the cache.
	Purge()
	// Refresh will crawl the cache and update their entries.
	// A timeout of 0 must mean no timeout.
	// Refresh should honor RefreshSleepTime for per-lookup
	// intervals unless the cache mechanism exposes its own
	// tunables.
	// Refresh may honor RefreshShuffle if it is practical or desirable.
	Refresh(timeout time.Duration)
	// Close should be used to signal end of operations.
	// The cache should be considered unusable after this.
	// Close may return an error, but should not assume it is consumed.
	Close() error
	// Add will upsert a collection into the cache.
	Add(address string, ips []net.IP)
	// Remove will remove a collection from the cache, if it exists.
	Remove(address string)
	// Get will return a collection from the cache, also bool if
	// a collection was retrieved.
	Get(address string) ([]net.IP, bool)
	// Len will return the number of items in the cache.
	// Eventually-consistent or lazy caches may return estimates.
	Len() int
}

// ResolverConfig is a common configuration structure for the Resolver.
type ResolverConfig struct {
	Cache               ResolverCache
	AutoRefreshInterval time.Duration
	AutoRefreshTimeout  time.Duration
}
