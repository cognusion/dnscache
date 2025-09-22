package dnscache

import (
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

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
		r := New(0)
		r.Lookup("dns.google.com")
		r.lock.RLock()
		defer r.lock.RUnlock()
		So(ipsTov4(r.cache["dns.google.com"]...), ShouldResemble, googs)
	})
}

func TestFetchLoadsValueFromTheCache(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a DNSCache entry is hand-crafted, and Fetch is called on it, the result is expected.", t, func() {
		r := New(0)
		r.cache["invalid.viki.io"] = stringsToIPs("1.1.2.3")
		ips, _ := r.Fetch("invalid.viki.io")
		So(ips, ShouldResemble, stringsToIPs("1.1.2.3"))
	})
}

func TestFetchOneLoadsTheFirstValue(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a DNSCache entry is hand-crafted, and FetchOne is called on it, the result is expected.", t, func() {
		r := New(0)
		r.cache["something.viki.io"] = stringsToIPs("1.1.2.3", "100.100.102.103")
		ip, _ := r.FetchOne("something.viki.io")
		So([]net.IP{ip}, ShouldResemble, stringsToIPs("1.1.2.3"))
	})
}

func TestFetchOneStringLoadsTheFirstValue(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a DNSCache entry is hand-crafted, and FetchOneString is called on it, the result is expected.", t, func() {
		r := New(0)
		r.cache["something.viki.io"] = stringsToIPs("100.100.102.103", "100.100.102.104")
		ip, _ := r.FetchOneString("something.viki.io")
		So(ip, ShouldEqual, "100.100.102.103")
	})
}

func TestFetchLoadsTheIpAndCachesIt(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a DNS entry is fetched, it is correct", t, func() {
		r := New(0)
		ips, _ := r.Fetch("dns.google.com")
		So(ipsTov4(ips...), ShouldResemble, googs)
		Convey("And so is the cache", func() {
			r.lock.RLock()
			defer r.lock.RUnlock()
			So(ipsTov4(r.cache["dns.google.com"]...), ShouldResemble, googs)
		})
	})
}

func TestItReloadsTheIpsAtAGivenInterval(t *testing.T) {
	defer leaktest.Check(t)()

	Convey("When a DNSCache is created with an autorefresh interval, and an entry is corrupted, it properly refreshes.", t, func() {
		RefreshSleepTime = 0 // Set this to immediate
		r := New(10 * time.Millisecond)
		defer r.Close() // if we're using autorefresh, Close prevents a goroleak.

		r.lock.Lock()
		r.cache["dns.google.com"] = nil
		r.lock.Unlock()
		time.Sleep(100 * time.Millisecond)
		r.lock.RLock()
		defer r.lock.RUnlock()
		So(ipsTov4(r.cache["dns.google.com"]...), ShouldResemble, googs)
	})
}

func TestAGTimeout(t *testing.T) {
	defer leaktest.Check(t)()

	// Some times these lose a race if testing with -race. I don't think they warrant a GL or atomic vars.
	RefreshSleepTime = 4 * time.Second // Set this to an unreasonably large number
	RefreshShuffle = false             // Turn off else unpredictable.

	Convey("When a DNSCache is created with an autorefresh interval, and an entry is corrupted, it properly refreshes.", t, FailureContinues, func() {
		r := NewWithRefreshTimeout(20*time.Millisecond, 1*time.Millisecond) // Set the timeout unreasonably small
		defer r.Close()                                                     // if we're using autorefresh, Close prevents a goroleak.

		r.lock.Lock()
		r.cache["dns.google.com"] = nil
		r.cache["www.google.com"] = nil
		r.cache["images.google.com"] = nil
		r.lock.Unlock()
		time.Sleep(100 * time.Millisecond) // 3-5 refresh runs
		r.lock.RLock()
		defer r.lock.RUnlock()
		So(ipsTov4(r.cache["dns.google.com"]...), ShouldResemble, googs) // first always gets a lookup
		So(r.cache["www.google.com"], ShouldBeZeroValue)                 // will always miss the timeout
		So(r.cache["images.google.com"], ShouldBeZeroValue)              // will always miss the timeout
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
