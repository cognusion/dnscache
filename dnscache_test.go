package dnscache

import (
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/cognusion/dnscache/cache"
	"github.com/fortytw2/leaktest"
	. "github.com/smartystreets/goconvey/convey"
)

var googs = []string{"8.8.4.4", "8.8.8.8"}

func ExampleNew() {
	//refresh items every 5 minutes
	resolver := New(time.Minute * 5)
	//get an array of net.IP
	ips, _ := resolver.Fetch("dns.google.com")
	fmt.Printf("%+v\n", ips)
}

func ExampleNewWithRefreshTimeout() {
	//refresh items every 5 minutes, timeout each refresh after 1 minute.
	resolver := NewWithRefreshTimeout(time.Minute*5, time.Minute*1)
	//get an array of net.IP
	ips, _ := resolver.Fetch("dns.google.com")
	fmt.Printf("%+v\n", ips)
}

func ExampleResolver_Fetch() {
	//refresh items every 5 minutes
	resolver := New(time.Minute * 5)
	//get an array of net.IP
	ips, _ := resolver.Fetch("dns.google.com")
	fmt.Printf("%+v\n", ips)
}

func ExampleResolver_FetchOne() {
	//refresh items every 5 minutes
	resolver := New(time.Minute * 5)
	//get the first net.IP
	ip, _ := resolver.FetchOne("dns.google.com")
	fmt.Printf("%+v\n", ip)
}

func ExampleResolver_FetchOneString() {
	//refresh items every 5 minutes
	resolver := New(time.Minute * 5)
	//get the first net.IP as string
	ipString, _ := resolver.FetchOneString("dns.google.com")
	fmt.Printf("%s\n", ipString)
}

// If you want to specify your cache style, then NewFromConfig is for you.
func ExampleNewFromConfig() {
	theCache, err := cache.NewLRU(
		cache.NewConfigOption(cache.ConfigSize, 100),                       // Keep up to 100 items in the cache
		cache.NewConfigOption(cache.ConfigRefreshType, cache.RefreshBatch), // Batch refreshes
		cache.NewConfigOption(cache.ConfigRefreshBatchSize, 10),            // Refresh in batches of 10
	)
	if err != nil {
		// don't actually do this.
		panic(err)
	}

	//refresh items every 5 minutes
	resolver := NewFromConfig(&ResolverConfig{
		Cache:               theCache,
		AutoRefreshInterval: 5 * time.Minute,
	})
	//get an array of net.IP
	ips, _ := resolver.Fetch("dns.google.com")
	fmt.Printf("%+v\n", ips)
}

// If you are using an `http.Transport`, you can use this cache by specifying a `Dial` function.
func Example() {
	// Create a resolver somewhere
	resolver := New(5 * time.Minute)

	transport := &http.Transport{
		MaxIdleConnsPerHost: 64,
		Dial: func(network string, address string) (net.Conn, error) {
			separator := strings.LastIndex(address, ":")
			ip, _ := resolver.FetchOneString(address[:separator])
			return net.Dial("tcp", ip+address[separator:])
		},
	}

	// e.g.
	http.DefaultTransport = transport
}

func TestFetchReturnsAndErrorOnInvalidLookup(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When an invalid lookup occurs, the expected results happen", t, func() {
		ips, err := New(0).Lookup("invalid.viki.io")
		So(ips, ShouldBeZeroValue)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldEqual, "lookup invalid.viki.io: no such host")
	})
}

func TestFetchReturnsAListOfIps(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a known lookup occurs, the expected results happen", t, func() {
		ips, _ := New(0).Lookup("dns.google.com")
		So(ipsTov4(ips...), ShouldResemble, googs)
	})
}

func TestCallingLookupAddsTheItemToTheCache(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a known lookup occurs, the expected results are in the raw cache", t, func() {
		c, err := cache.NewSimple()
		So(err, ShouldBeNil)

		r := NewFromConfig(&ResolverConfig{
			Cache: c,
		})
		r.Lookup("dns.google.com")

		ips, ok := c.Get("dns.google.com")
		So(ok, ShouldBeTrue)
		So(ipsTov4(ips...), ShouldResemble, googs)

		Convey("When the cache is purged, it is empty", func() {
			SoMsg("Expected 1 item is not in cache", c.Len(), ShouldEqual, 1)
			r.Purge()
			So(c.Len(), ShouldEqual, 0)
		})
	})
}

func TestFetchLoadsValueFromTheCache(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a DNSCache entry is hand-crafted, and Fetch is called on it, the result is expected.", t, func() {
		c, err := cache.NewSimple()
		So(err, ShouldBeNil)

		r := NewFromConfig(&ResolverConfig{
			Cache: c,
		})
		c.Add("invalid.viki.io", stringsToIPs("1.1.2.3"))
		ips, _ := r.Fetch("invalid.viki.io")
		So(ips, ShouldResemble, stringsToIPs("1.1.2.3"))
	})
}

func TestFetchOneLoadsTheFirstValue(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a DNSCache entry is hand-crafted, and FetchOne is called on it, the result is expected.", t, func() {
		c, err := cache.NewSimple()
		So(err, ShouldBeNil)

		r := NewFromConfig(&ResolverConfig{
			Cache: c,
		})
		c.Add("something.viki.io", stringsToIPs("1.1.2.3", "100.100.102.103"))
		ip, _ := r.FetchOne("something.viki.io")
		So([]net.IP{ip}, ShouldResemble, stringsToIPs("1.1.2.3"))
	})
}

func TestFetchOneStringLoadsTheFirstValue(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a DNSCache entry is hand-crafted, and FetchOneString is called on it, the result is expected.", t, func() {
		c, err := cache.NewSimple()
		So(err, ShouldBeNil)

		r := NewFromConfig(&ResolverConfig{
			Cache: c,
		})
		c.Add("something.viki.io", stringsToIPs("100.100.102.103", "100.100.102.104"))
		ip, _ := r.FetchOneString("something.viki.io")
		So(ip, ShouldEqual, "100.100.102.103")
	})
}

func TestFetchLoadsTheIpAndCachesIt(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a DNS entry is fetched, it is correct", t, func() {
		c, err := cache.NewSimple()
		So(err, ShouldBeNil)

		r := NewFromConfig(&ResolverConfig{
			Cache: c,
		})
		ips, _ := r.Fetch("dns.google.com")
		So(ipsTov4(ips...), ShouldResemble, googs)
		Convey("And so is the cache", func() {
			ips, ok := c.Get("dns.google.com")
			So(ok, ShouldBeTrue)
			So(ipsTov4(ips...), ShouldResemble, googs)
		})
	})
}

func TestNewFromCacheNilCache(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a DNSCache is created with a config, but the cache is nil, it works as expected up.", t, func() {
		r := NewFromConfig(&ResolverConfig{})
		So(r, ShouldNotBeNil)

		ips, _ := r.Fetch("dns.google.com")
		So(ipsTov4(ips...), ShouldResemble, googs)
	})
}

func TestItReloadsTheIpsAtAGivenInterval(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a DNSCache is created with an autorefresh interval, and an entry is corrupted, it properly refreshes.", t, func() {
		c, err := cache.NewSimple(
			cache.NewConfigOption(cache.ConfigRefreshSleepTime, time.Duration(0)), // immediate
			cache.NewConfigOption(cache.ConfigRefreshShuffle, false),              // else unpredictable
		)
		So(err, ShouldBeNil)

		r := NewFromConfig(&ResolverConfig{
			Cache:               c,
			AutoRefreshInterval: 10 * time.Millisecond,
		})
		defer r.Close() // if we're using autorefresh, Close prevents a goroleak.

		c.Add("dns.google.com", []net.IP{})
		time.Sleep(100 * time.Millisecond)

		ips, ok := c.Get("dns.google.com")
		So(ok, ShouldBeTrue)
		So(ipsTov4(ips...), ShouldResemble, googs)
	})

	Convey("When an LRU DNSCache is created with an autorefresh interval, and an entry is corrupted, it properly refreshes.", t, func() {
		c, err := cache.NewLRU(
			cache.NewConfigOption(cache.ConfigRefreshSleepTime, time.Duration(0)), // immediate
			cache.NewConfigOption(cache.ConfigRefreshShuffle, false),              // else unpredictable
			cache.NewConfigOption(cache.ConfigSize, 5),
		)
		So(err, ShouldBeNil)

		r := NewFromConfig(&ResolverConfig{
			Cache:               c,
			AutoRefreshInterval: 10 * time.Millisecond,
		})
		defer r.Close() // if we're using autorefresh, Close prevents a goroleak.

		c.Add("dns.google.com", []net.IP{})
		c.Add("images.google.com", []net.IP{})
		time.Sleep(100 * time.Millisecond)
		ips, ok := c.Get("dns.google.com")
		So(ok, ShouldBeTrue)
		So(ipsTov4(ips...), ShouldResemble, googs)
		ips, ok = c.Get("images.google.com")
		So(ok, ShouldBeTrue)
		So(ips, ShouldNotBeZeroValue)

		fips, ferr := r.Fetch("dns.google.com")
		So(ferr, ShouldBeNil)
		So(ipsTov4(fips...), ShouldResemble, googs)
	})
}

func TestAGTimeout(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a DNSCache is created with an autorefresh interval, and an entry is corrupted, it properly refreshes.", t, FailureContinues, func() {
		c, err := cache.NewSimple(
			cache.NewConfigOption(cache.ConfigRefreshSleepTime, 4*time.Second), // loooooong time
			cache.NewConfigOption(cache.ConfigRefreshShuffle, false),           // else unpredictable
		)
		So(err, ShouldBeNil)

		r := NewFromConfig(&ResolverConfig{
			Cache:               c,
			AutoRefreshInterval: 20 * time.Millisecond,
			AutoRefreshTimeout:  1 * time.Millisecond,
		})
		defer r.Close() // if we're using autorefresh, Close prevents a goroleak.

		c.Add("dns.google.com", []net.IP{})
		c.Add("www.google.com", []net.IP{})
		c.Add("images.google.com", []net.IP{})

		time.Sleep(100 * time.Millisecond) // 3-5 refresh runs

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

func TestResolverCaches(t *testing.T) {
	Convey("The provided caches implement ResolverCache", t, func() {
		r := &cache.Simple{}
		SoMsg("cache.Simple is no longer a ResolverCache!", r, ShouldImplement, (*ResolverCache)(nil))
		l := &cache.LRU{}
		SoMsg("cache,LRU is no longer a ResolverCache!", l, ShouldImplement, (*ResolverCache)(nil))
	})
}

func TestEmptyCacheRefresh(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a DNSCache is created, and forcibly refreshed despite being empty, nothing explodes, and the Refresh occurs quickly.", t, func() {
		r := New(0) // no autorefresh
		start := time.Now()
		r.Refresh()
		after := time.Now()
		So(after, ShouldHappenWithin, 10*time.Millisecond, start)
	})
}

func stringsToIPs(strs ...string) []net.IP {
	ips := make([]net.IP, len(strs))
	for i, s := range strs {
		ips[i] = net.ParseIP(s).To4()
	}
	return ips
}

func ipsTov4(ips ...net.IP) []string {
	ip4s := make([]string, 0)
	for _, i := range ips {
		i4 := i.To4()
		if i4 != nil {
			ip4s = append(ip4s, i4.String())
		}
	}
	slices.Sort(ip4s)
	return ip4s
}
