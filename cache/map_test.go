package cache

import (
	"net"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	. "github.com/smartystreets/goconvey/convey"
)

var googs = []string{"8.8.4.4", "8.8.8.8"}

func Test_SimpleInvalidLookup(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a Simple is created and an invalid lookup occurs, the expected results happen", t, func() {
		c, err := NewSimple()
		So(err, ShouldBeNil)
		defer c.Close()

		ips, err := c.Lookup("invalid.viki.io")
		So(ips, ShouldBeZeroValue)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "lookup invalid.viki.io: no such host")
	})
}

func Test_SimpleFetchLenPurge(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a Simple is created and a known-good fetch occurs, the expected results are returned, and in the cache", t, func() {
		c, err := NewSimple()
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

func Test_SimpleRefresh(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a Simple is created and an entry is corrupted, it properly refreshes on-demand.", t, func() {
		c, err := NewSimple(
			NewConfigOption(ConfigRefreshSleepTime, time.Duration(0)), // immediate
			NewConfigOption(ConfigRefreshShuffle, false),              // else unpredictable
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

func Test_SimpleConfigOptions(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a Simple is created with all of the valid options, nothing explodes.", t, FailureContinues, func() {
		c, err := NewSimple(
			NewConfigOption(ConfigRefreshSleepTime, 4*time.Second),
			NewConfigOption(ConfigRefreshShuffle, false),
			NewConfigOption(ConfigResolver, DefaultResolver),
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
		})
	})
}

func Test_SimpleRefreshTimeout(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a Simple is created and a series of entries are corrupted corrupted, but the timeout too low, there are coldspots.", t, FailureContinues, func() {
		c, err := NewSimple(
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

func Test_SimpleEmptyCacheRefresh(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a Simple is created, and forcibly refreshed despite being empty, nothing explodes, and the Refresh occurs quickly.", t, func() {
		c, err := NewSimple()
		So(err, ShouldBeNil)
		defer c.Close()

		start := time.Now()
		c.Refresh(0)
		after := time.Now()
		So(after, ShouldHappenWithin, 10*time.Millisecond, start)
	})
}
