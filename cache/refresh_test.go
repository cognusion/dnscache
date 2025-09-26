package cache

import (
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	. "github.com/smartystreets/goconvey/convey"
)

func benchSetup() (time.Duration, int) {
	// refreshSleep, number of cache items
	return 0 * time.Millisecond, 1024
}

func Test_RefreshLinear(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a LinearRefresh is ordered on a corrupted cache, all items are corrected.", t, func() {
		c, err := NewSimple(
			NewConfigOption(ConfigRefreshSleepTime, time.Duration(0)), // immediate
			NewConfigOption(ConfigRefreshShuffle, false),              // else unpredictable
		)
		So(err, ShouldBeNil)
		defer c.Close()

		for i := range 1000 {
			c.Add(fmt.Sprintf("%d.localhost", i), []net.IP{})
		}
		c.Refresh(0) // force a refresh, waiting

		for i := range 1000 {
			ips, ok := c.Get(fmt.Sprintf("%d.localhost", i))
			So(ok, ShouldBeTrue)
			So(ipsTov4(ips...), ShouldResemble, []string{"127.0.0.1"})
		}
	})
}

func Test_RefreshBatch(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a BatchRefresh is ordered on a corrupted cache, all items are corrected.", t, func() {
		c, err := NewSimple(
			NewConfigOption(ConfigRefreshSleepTime, time.Duration(0)), // immediate
			NewConfigOption(ConfigRefreshShuffle, false),              // else unpredictable
			NewConfigOption(ConfigRefreshType, RefreshBatch),          // batch
			NewConfigOption(ConfigRefreshBatchSize, 15),               // 15 at a time
		)

		So(err, ShouldBeNil)
		defer c.Close()

		for i := range 1000 {
			c.Add(fmt.Sprintf("%d.localhost", i), []net.IP{})
		}
		c.Refresh(0) // force a refresh, waiting

		for i := range 1000 {
			ips, ok := c.Get(fmt.Sprintf("%d.localhost", i))
			So(ok, ShouldBeTrue)
			So(ipsTov4(ips...), ShouldResemble, []string{"127.0.0.1"})
		}
	})
}

func Test_Shuffle(t *testing.T) {
	Convey("When a pair of consistent string slices are created and each shuffled, they are sufficiently different", t, func() {
		itemCount := 500
		addresses := make([]string, itemCount)
		addresses1 := make([]string, itemCount)
		addresses2 := make([]string, itemCount)
		var s string
		for i := range itemCount {
			s = fmt.Sprintf("%d.localhost", i)
			addresses[i] = s
			addresses1[i] = s
			addresses2[i] = s
		}

		rand.Shuffle(len(addresses1), func(i, j int) {
			addresses1[i], addresses1[j] = addresses1[j], addresses1[i]
		})

		rand.Shuffle(len(addresses2), func(i, j int) {
			addresses2[i], addresses2[j] = addresses2[j], addresses2[i]
		})

		SoMsg("Shuffled slice resembles unshuffled slice", addresses, ShouldNotResemble, addresses1)
		SoMsg("Shuffled slice resembles unshuffled slice", addresses, ShouldNotResemble, addresses2)
		SoMsg("Shuffled slice resembles other shuffled slice", addresses1, ShouldNotResemble, addresses2)
	})

}

// Benchmark notes.
// * Obviously different size batches elicit different performances
// * Performance differences between LRUs and Simple caches are similar.
//    * LRUs are slightly less performant in most cases due to bookkeeping overhead.
// * Obviously injecting RefreshSleepTime slows everything down, especially the linears (benchmarks set it to 0)
//
// All three of the refreshers make ~1-1.5kB of memory allocation per cache item, consistent from N=1 to 1024
// 	RefreshLinear:		1552B-	/item
//	RefreshLinearOld:	1208B	/item
//	RefreshBatch:		1053B+	/item
// For trivial caches RefreshBatch is the fastest, and RefreshLinear&Old are the slowest:
//	RefreshLinear:		78.721us-	/item
// 	RefreshLinearOld:	78.495us	/item
// 	RefreshBatch:		3.140us+	/item
// For large caches, the Linears stay consistent, but Batch slows slightly, still much more performant:
//	RefreshLinear:		78.530us	/item
// 	RefreshLinearOld:	79.298us-	/item
//	RefreshBatch:		27.511us+	/item

func Benchmark_RefreshLinear(b *testing.B) {
	refreshSleep, itemCount := benchSetup()

	c, err := NewSimple(
		NewConfigOption(ConfigRefreshSleepTime, time.Duration(refreshSleep)),
		NewConfigOption(ConfigRefreshShuffle, false), // else unpredictable
		//NewConfigOption(ConfigSize, itemCount*2),
	)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	for i := range itemCount {
		c.Add(fmt.Sprintf("%d.localhost", i), []net.IP{})
	}
	c.Refresh(0) // prime it

	b.ResetTimer()
	for b.Loop() {
		c.Refresh(0)
	}
}

func Benchmark_RefreshLinearOld(b *testing.B) {
	refreshSleep, itemCount := benchSetup()

	// create a fake cache
	var lock sync.RWMutex
	var cache = make(map[string][]net.IP, itemCount)
	for i := range itemCount {
		cache[fmt.Sprintf("%d.localhost", i)] = []net.IP{}
	}

	// Old-style lookup
	lu := func(address string) ([]net.IP, error) {
		ips, err := net.LookupIP(address)
		if err != nil {
			return nil, err
		}

		lock.Lock()
		cache[address] = ips
		lock.Unlock()
		return ips, nil
	}

	// Old-style refresh
	rf := func() {
		i := 0
		lock.RLock()
		addresses := make([]string, len(cache))
		for key := range cache {
			addresses[i] = key
			i++
		}
		lock.RUnlock()

		for _, address := range addresses {
			lu(address)
			time.Sleep(refreshSleep)
		}
	}

	rf() // prime it

	b.ResetTimer()
	for b.Loop() {
		rf()
	}

}

func Benchmark_RefreshBatch(b *testing.B) {
	refreshSleep, itemCount := benchSetup()

	c, err := NewSimple(
		NewConfigOption(ConfigRefreshSleepTime, time.Duration(refreshSleep)),
		NewConfigOption(ConfigRefreshShuffle, false),     // else unpredictable
		NewConfigOption(ConfigRefreshType, RefreshBatch), // batch
		NewConfigOption(ConfigRefreshBatchSize, 30),
		//NewConfigOption(ConfigSize, itemCount*2),
	)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	for i := range itemCount {
		c.Add(fmt.Sprintf("%d.localhost", i), []net.IP{})
	}
	c.Refresh(0) // prime it

	b.ResetTimer()
	for b.Loop() {
		c.Refresh(0)
	}
}

func Benchmark_Shuffle10(b *testing.B) {
	itemCount := 10
	addresses := make([]string, itemCount)
	for i := range itemCount {
		addresses[i] = fmt.Sprintf("%d.localhost", i)
	}

	b.ResetTimer()
	for b.Loop() {
		rand.Shuffle(len(addresses), func(i, j int) {
			addresses[i], addresses[j] = addresses[j], addresses[i]
		})
	}
}

func Benchmark_Shuffle100(b *testing.B) {
	itemCount := 100
	addresses := make([]string, itemCount)
	for i := range itemCount {
		addresses[i] = fmt.Sprintf("%d.localhost", i)
	}

	b.ResetTimer()
	for b.Loop() {
		rand.Shuffle(len(addresses), func(i, j int) {
			addresses[i], addresses[j] = addresses[j], addresses[i]
		})
	}
}

func Benchmark_Shuffle1000(b *testing.B) {
	itemCount := 1000
	addresses := make([]string, itemCount)
	for i := range itemCount {
		addresses[i] = fmt.Sprintf("%d.localhost", i)
	}

	b.ResetTimer()
	for b.Loop() {
		rand.Shuffle(len(addresses), func(i, j int) {
			addresses[i], addresses[j] = addresses[j], addresses[i]
		})
	}
}
