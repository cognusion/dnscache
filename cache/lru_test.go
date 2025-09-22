package cache

import (
	"net"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	. "github.com/smartystreets/goconvey/convey"
)

// var googs = []string{"8.8.4.4", "8.8.8.8"}
const ConfigNotAnOption = ConfigKey("NotAnOption")

func Test_LRUMissingRequiredOption(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When an LRU is created without a required option, an error is returned.", t, func() {
		c, err := NewLRU()
		So(err, ShouldBeError)
		So(c, ShouldBeNil)
	})

	Convey("When an LRU is created with required option but wrong value type, an error is returned.", t, func() {
		c, err := NewLRU(
			NewConfigOption(ConfigSize, false),
		)
		So(err, ShouldBeError)
		So(c, ShouldBeNil)
	})
}

func Test_LRUInvalidLookup(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When an LRU is created and an invalid lookup occurs, the expected results happen", t, func() {
		c, err := NewLRU(
			NewConfigOption(ConfigSize, 10),
		)
		So(err, ShouldBeNil)
		defer c.Close()

		ips, err := c.Lookup("invalid.viki.io")
		So(ips, ShouldBeZeroValue)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "lookup invalid.viki.io: no such host")
	})
}

func Test_LRUFetchLenPurge(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When an LRU is created and a known-good fetch occurs, the expected results are returned, and in the cache", t, func() {
		c, err := NewLRU(
			NewConfigOption(ConfigSize, 10),
		)
		So(err, ShouldBeNil)
		defer c.Close()

		ips, err := c.Fetch("dns.google.com")
		So(err, ShouldBeNil)
		So(ipsTov4(ips...), ShouldResemble, googs)

		ips, ok := c.Get("dns.google.com")
		So(ok, ShouldBeTrue)
		So(ipsTov4(ips...), ShouldResemble, googs)

		ips, err = c.Fetch("dns.google.com")
		So(err, ShouldBeNil)
		So(ipsTov4(ips...), ShouldResemble, googs)

		Convey("When the cache is purged, it is empty", func() {
			SoMsg("Expected 1 item is not in cache", c.Len(), ShouldEqual, 1)
			c.Purge()
			So(c.Len(), ShouldEqual, 0)
		})
	})
}

func Test_ExpirableLRUFetchLenPurge(t *testing.T) {

	// Cannot leaktest expirable.LRU. https://github.com/hashicorp/golang-lru/blob/1ecdc13547b564bf736db9161ed89f1864010108/expirable/expirable_lru.go#L53
	Convey("When an expirable LRU is created and a known-good fetch occurs, the expected results are returned, and in the cache", t, func() {
		c, err := NewLRU(
			NewConfigOption(ConfigSize, 10),
			NewConfigOption(ConfigItemTTL, 1*time.Minute),
		)
		So(err, ShouldBeNil)
		defer c.Close()

		ips, err := c.Fetch("dns.google.com")
		So(err, ShouldBeNil)
		So(ipsTov4(ips...), ShouldResemble, googs)

		ips, ok := c.Get("dns.google.com")
		So(ok, ShouldBeTrue)
		So(ipsTov4(ips...), ShouldResemble, googs)

		ips, err = c.Fetch("dns.google.com")
		So(err, ShouldBeNil)
		So(ipsTov4(ips...), ShouldResemble, googs)

		Convey("When the cache is purged, it is empty", func() {
			SoMsg("Expected 1 item is not in cache", c.Len(), ShouldEqual, 1)
			c.Purge()
			So(c.Len(), ShouldEqual, 0)
		})
	})
}

func Test_LRURefresh(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When an LRU is created and an entry is corrupted, it properly refreshes on-demand.", t, func() {
		c, err := NewLRU(
			NewConfigOption(ConfigSize, 10),
			NewConfigOption(ConfigRefreshSleepTime, time.Duration(0)), // immediate
		)
		So(err, ShouldBeNil)
		defer c.Close()

		c.Add("dns.google.com", []net.IP{})
		c.Refresh(0) // force a refresh, waiting
		ips, ok := c.Get("dns.google.com")
		So(ok, ShouldBeTrue)
		So(ipsTov4(ips...), ShouldResemble, googs)

		Convey("When the entry is removed, manually, the cache is empty", func() {
			SoMsg("Expected 1 item is not in cache", c.Len(), ShouldEqual, 1)
			c.Remove("dns.google.com")
			So(c.Len(), ShouldEqual, 0)
		})
	})
}

func Test_LRUConfigOptions(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When an LRU is created with all of the valid options, nothing explodes.", t, func() {
		c, err := NewLRU(
			NewConfigOption(ConfigSize, 10),
			NewConfigOption(ConfigRefreshSleepTime, 4*time.Second),
			NewConfigOption(ConfigRefreshShuffle, false),
			NewConfigOption(ConfigResolver, DefaultResolver),
			NewConfigOption(ConfigAllowRefresh, false),
		)
		So(err, ShouldBeNil)
		So(c, ShouldNotBeNil)
		defer c.Close()

		Convey("When an invalid option is passed, it generates an appropriate error", func() {
			So(c.config(NewConfigOption(ConfigNotAnOption, 5)), ShouldEqual, ErrorConfigKeyUnsupported)
		})

		Convey("When a valid option with an invalid type is passed, it generates an appropriate error", func() {
			So(c.config(NewConfigOption(ConfigRefreshShuffle, 16)), ShouldBeError)
			So(c.config(NewConfigOption(ConfigRefreshSleepTime, []string{})), ShouldBeError)
			So(c.config(NewConfigOption(ConfigResolver, 42)), ShouldBeError)
			So(c.config(NewConfigOption(ConfigAllowRefresh, 42)), ShouldBeError)

		})
	})

}

func Test_ExpirableLRUConfigOptions(t *testing.T) {

	// Cannot leaktest expirable.LRU. https://github.com/hashicorp/golang-lru/blob/1ecdc13547b564bf736db9161ed89f1864010108/expirable/expirable_lru.go#L53
	Convey("When an LRU is created with all of the valid options, nothing explodes.", t, func() {
		c, err := NewLRU(
			NewConfigOption(ConfigSize, 10),
			NewConfigOption(ConfigItemTTL, 1*time.Minute),
			NewConfigOption(ConfigRefreshSleepTime, 4*time.Second),
			NewConfigOption(ConfigRefreshShuffle, false),
			NewConfigOption(ConfigResolver, DefaultResolver),
			NewConfigOption(ConfigAllowRefresh, false),
		)
		So(err, ShouldBeNil)
		So(c, ShouldNotBeNil)
		defer c.Close()

		Convey("When an invalid option is passed, it generates an appropriate error", func() {
			So(c.config(NewConfigOption(ConfigNotAnOption, 5)), ShouldEqual, ErrorConfigKeyUnsupported)
		})

		Convey("When a valid option with an invalid type is passed, it generates an appropriate error", func() {
			So(c.config(NewConfigOption(ConfigRefreshShuffle, 16)), ShouldBeError)
			So(c.config(NewConfigOption(ConfigRefreshSleepTime, []string{})), ShouldBeError)
			So(c.config(NewConfigOption(ConfigResolver, 42)), ShouldBeError)
			So(c.config(NewConfigOption(ConfigAllowRefresh, 42)), ShouldBeError)
			So(c.config(NewConfigOption(ConfigSize, nil)), ShouldBeError)
			So(c.config(NewConfigOption(ConfigItemTTL, "hello world")), ShouldBeError)

		})
	})

}

func Test_LRURefreshTimeout(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When an LRU is created and a series of entries are corrupted corrupted, but the timeout too low, there are coldspots.", t, func() {
		c, err := NewLRU(
			NewConfigOption(ConfigSize, 10),
			NewConfigOption(ConfigRefreshSleepTime, 4*time.Second), // loooooong time
			NewConfigOption(ConfigRefreshShuffle, false),           // else unpredictable
		)
		So(err, ShouldBeNil)
		defer c.Close()

		c.Add("dns.google.com", []net.IP{})
		c.Add("www.google.com", []net.IP{})
		c.Add("images.google.com", []net.IP{})

		c.Refresh(1 * time.Millisecond)

		ips, ok := c.Get("dns.google.com")
		So(ok, ShouldBeTrue)
		So(ipsTov4(ips...), ShouldResemble, googs) // first always gets a lookup
		ips, ok = c.Get("www.google.com")
		So(ok, ShouldBeTrue)
		So(ips, ShouldBeEmpty) // should always miss
		ips, ok = c.Get("images.google.com")
		So(ok, ShouldBeTrue)
		So(ips, ShouldBeEmpty) // should always miss
	})
}

func Test_LRUEmptyCacheRefresh(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When an LRU is created, and forcibly refreshed despite being empty, nothing explodes, and the Refresh occurs quickly.", t, func() {
		c, err := NewLRU(
			NewConfigOption(ConfigSize, 10),
		)
		So(err, ShouldBeNil)
		defer c.Close()

		start := time.Now()
		c.Refresh(0)
		after := time.Now()
		So(after, ShouldHappenWithin, 10*time.Millisecond, start)
	})
}
