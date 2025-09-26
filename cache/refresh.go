package cache

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"
)

const (
	// ConfigRefreshType is a RefreshType.
	// The RefreshTypes are defined elsewhere.
	ConfigRefreshType = ConfigKey("RefreshType")
	// ConfigRefreshBatchSize is an int.
	// This is the number of lookups to process per batch, when
	// a Batch-type RefreshType is used.
	// Values below 5 are untestest and unlikely to be useful.
	ConfigRefreshBatchSize = ConfigKey("RefreshBatchSize")
	// ConfigRefreshTimeout is a time.Duration.
	// For values > 0, this is treated as a per-loop deadline
	// to complete a Refresh.
	ConfigRefreshTimeout = ConfigKey("RefreshTimeout")

	// RefreshOff is a RefreshType used when the cache should silently refuse
	// to do Refreshes if requested.
	RefreshOff = RefreshType("RefreshOff")
	// RefreshLinear is a RefreshType used for small caches or on small systems
	// where grinding lookups might become a problem.
	RefreshLinear = RefreshType("RefreshLinear")
	// RefreshBatch is a RefreshType that spawns lookups in flights up to `ConfigRefreshBatchSize`
	// every iteration. It is the most performant option for large caches, and is also well-suited
	// for anything but the smallest of systems.
	RefreshBatch = RefreshType("RefreshBatch")
)

// NoRefresh is a noop RefreshFunc that always returns true, and never an error.
func NoRefresh(cache RefreshableCache, resolver ResolverFunc, options ...ConfigOption) (bool, error) {
	return true, nil
}

// LinearRefresh is the classic ordered, one-at-a-time RefreshFunc. By default, it will shuffle the keys,
// sleep for 1s between each lookup, and continue until it is done (no timeout).
func LinearRefresh(cache RefreshableCache, resolver ResolverFunc, options ...ConfigOption) (bool, error) {
	var (
		refreshShuffle   bool          = true
		refreshSleepTime time.Duration = 1 * time.Second
		refreshTimeout   time.Duration // default off
	)
	for _, o := range options {
		switch o.Key {
		case ConfigRefreshShuffle:
			if v, ok := o.Value.(bool); ok {
				refreshShuffle = v
			} else {
				return false, o.Key.Error()
			}
		case ConfigRefreshSleepTime:
			if v, ok := o.Value.(time.Duration); ok {
				refreshSleepTime = v
			} else {
				return false, o.Key.Error()
			}
		case ConfigRefreshTimeout:
			if v, ok := o.Value.(time.Duration); ok {
				refreshTimeout = v
			} else {
				return false, o.Key.Error()
			}
		default:
			return false, ErrorConfigKeyUnsupported
		}
	}

	// Get the keys
	addresses := cache.Keys()

	if len(addresses) == 0 {
		// empty cache
		return true, nil
	}

	if refreshShuffle {
		rand.Shuffle(len(addresses), func(i, j int) {
			addresses[i], addresses[j] = addresses[j], addresses[i]
		})
	}

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	if refreshTimeout == 0 {
		// No deadline
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		// Deadline
		ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(refreshTimeout))
	}
	defer cancel() // because yes

	// first lookup is out of loop, so we don't wait
	resolver(addresses[0])

	// offset i to account for the previous lookup
	for i := 1; i < len(addresses); i++ {
		select {
		case <-time.After(refreshSleepTime):
			// this loop is here because it is highly possible that one or more of the
			// previously-existing addresses no longer is in the cache, due to
			// pressure or TTL evictions. So we peek into the cache to see if an
			// address still exists, until one finally does, then we break out and
			// outer-loop again.
		STALE:
			for {
				if i >= len(addresses) {
					// that's all folks
					return true, nil
				}
				if cache.Contains(addresses[i]) {
					resolver(addresses[i])
					break STALE
				}
				i++
			}
		case <-ctx.Done():
			// took too long, deadline exceeded.
			return false, nil
		}
	}
	return true, nil
}

// BatchRefresh uses workers to do RefreshBatchSize lookups at a time. By default, it will shuffle the keys,
// sleep 1s between each batch, and run until it is done (no timeout).
func BatchRefresh(cache RefreshableCache, resolver ResolverFunc, options ...ConfigOption) (bool, error) {
	var (
		refreshShuffle   bool          = true
		refreshSleepTime time.Duration = 1 * time.Second
		refreshTimeout   time.Duration // default off
		batchSize        int
	)
	if v, ok := ConfigRefreshBatchSize.IsIn(options); !ok {
		return false, fmt.Errorf("option %s is required", ConfigRefreshBatchSize)
	} else if batchSize, ok = v.(int); !ok {
		return false, ConfigRefreshBatchSize.Error()
	}

	for _, o := range options {
		switch o.Key {
		case ConfigRefreshShuffle:
			if v, ok := o.Value.(bool); ok {
				refreshShuffle = v
			} else {
				return false, o.Key.Error()
			}
		case ConfigRefreshSleepTime:
			if v, ok := o.Value.(time.Duration); ok {
				refreshSleepTime = v
			} else {
				return false, o.Key.Error()
			}
		case ConfigRefreshTimeout:
			if v, ok := o.Value.(time.Duration); ok {
				refreshTimeout = v
			} else {
				return false, o.Key.Error()
			}
		case ConfigRefreshBatchSize:
			// we already applied this.
		default:
			return false, ErrorConfigKeyUnsupported
		}
	}

	// Get the keys
	addresses := cache.Keys()

	if len(addresses) == 0 {
		// empty cache
		return true, nil
	}

	if refreshShuffle {
		rand.Shuffle(len(addresses), func(i, j int) {
			addresses[i], addresses[j] = addresses[j], addresses[i]
		})
	}

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	if refreshTimeout == 0 {
		// No deadline
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		// Deadline
		ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(refreshTimeout))
	}
	defer cancel() // because yes

	var wg sync.WaitGroup

	wgResolver := func(a string) {
		defer wg.Done()
		resolver(a)
	}

	var total int
	for _, a := range addresses {
		wg.Add(1)
		go wgResolver(a)
		total++
		if total >= batchSize {
			break
		}
	}
	if total >= len(addresses) {
		//sub-batch cache size, we done!
		return true, nil
	}

	var maybe bool

DONE:
	for {
		select {
		case <-time.After(refreshSleepTime):
			run := 0
		STALE:
			for i := total; i < len(addresses); i++ {
				if cache.Contains(addresses[i]) {
					wg.Add(1)
					go wgResolver(addresses[i])
					// we will loop until we actually hit batchSize number,
					// accounting for those that have evicted along the way.
					run++
				}
				total++
				if total >= len(addresses) {
					maybe = true
					break DONE
				}
				if run >= batchSize {
					break STALE
				}
			}
		case <-ctx.Done():
			// took too long, deadline exceeded.
			return false, nil
		}
	}
	wg.Wait()
	return maybe, nil
}
