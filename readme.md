

# dnscache
`import "github.com/cognusion/dnscache"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)
* [Examples](#pkg-examples)
* [Subdirectories](#pkg-subdirectories)

## <a name="pkg-overview">Overview</a>
Package dnscache caches DNS lookups.
The package itself requires no non-standard modules, however
a separate testing suite is used.

Based on <a href="https://github.com/viki-org/dnscache">https://github.com/viki-org/dnscache</a> with modern Go
facilities, no intrinsic goro leak, more flexibility, and more.


##### Example :
If you are using an `http.Transport`, you can use this cache by specifying a `Dial` function.

``` go
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
```



## <a name="pkg-index">Index</a>
* [Variables](#pkg-variables)
* [type Resolver](#Resolver)
  * [func New(refreshRate time.Duration) *Resolver](#New)
  * [func NewFromConfig(config *ResolverConfig) *Resolver](#NewFromConfig)
  * [func NewWithRefreshTimeout(refreshRate, refreshTimeout time.Duration) *Resolver](#NewWithRefreshTimeout)
  * [func (r *Resolver) Close() error](#Resolver.Close)
  * [func (r *Resolver) Fetch(address string) ([]net.IP, error)](#Resolver.Fetch)
  * [func (r *Resolver) FetchOne(address string) (net.IP, error)](#Resolver.FetchOne)
  * [func (r *Resolver) FetchOneString(address string) (string, error)](#Resolver.FetchOneString)
  * [func (r *Resolver) Lookup(address string) ([]net.IP, error)](#Resolver.Lookup)
  * [func (r *Resolver) Purge()](#Resolver.Purge)
  * [func (r *Resolver) Refresh()](#Resolver.Refresh)
  * [func (r *Resolver) RefreshTimeout(timeout time.Duration)](#Resolver.RefreshTimeout)
* [type ResolverCache](#ResolverCache)
* [type ResolverConfig](#ResolverConfig)

#### <a name="pkg-examples">Examples</a>
* [Package](#example-)
* [New](#example-new)
* [NewWithRefreshTimeout](#example-newwithrefreshtimeout)
* [Resolver.Fetch](#example-resolver_fetch)
* [Resolver.FetchOne](#example-resolver_fetchone)
* [Resolver.FetchOneString](#example-resolver_fetchonestring)

#### <a name="pkg-files">Package files</a>
[cache.go](https://github.com/cognusion/dnscache/tree/master/cache.go) [dnscache.go](https://github.com/cognusion/dnscache/tree/master/dnscache.go)



## <a name="pkg-variables">Variables</a>
``` go
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
```



## <a name="Resolver">type</a> [Resolver](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=877:969#L30)
``` go
type Resolver struct {
    // contains filtered or unexported fields
}

```
Resolver is a goro-safe caching DNS resolver.







### <a name="New">func</a> [New](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=1118:1163#L39)
``` go
func New(refreshRate time.Duration) *Resolver
```
New returns a properly instantiated Resolver.
If the refreshRate is non-zero, a goro will refresh
all of the entries after that Duration.


##### Example New:
``` go
//refresh items every 5 minutes
resolver := New(time.Minute * 5)
//get an array of net.IP
ips, _ := resolver.Fetch("dns.google.com")
fmt.Printf("%+v\n", ips)
```

### <a name="NewFromConfig">func</a> [NewFromConfig](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=2305:2357#L69)
``` go
func NewFromConfig(config *ResolverConfig) *Resolver
```
NewFromConfig returns a properly instantiated resolver, using the provided Cache
and the provided AutoRefresh* values.
NOTE: If using an LRU-style cache, setting the AutoRefreshInterval has large as
feasible is advised, to keep the cache calculus correct.


### <a name="NewWithRefreshTimeout">func</a> [NewWithRefreshTimeout](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=1515:1594#L48)
``` go
func NewWithRefreshTimeout(refreshRate, refreshTimeout time.Duration) *Resolver
```
NewWithRefreshTimeout returns a properly instantiated Resolver.
If the refreshRate is non-zero, a goro will refresh
all of the entries after that Duration.
If refreshTimeout is non-zero, each auto-refresh iteraction will timeout after
the specified duration (expressed as a deadline).


##### Example NewWithRefreshTimeout:
``` go
//refresh items every 5 minutes, timeout each refresh after 1 minute.
resolver := NewWithRefreshTimeout(time.Minute*5, time.Minute*1)
//get an array of net.IP
ips, _ := resolver.Fetch("dns.google.com")
fmt.Printf("%+v\n", ips)
```




### <a name="Resolver.Close">func</a> (\*Resolver) [Close](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=2763:2795#L85)
``` go
func (r *Resolver) Close() error
```
Close signals the auto-refresh goro, if any, to quit.
This is safe to call once, in any thread, regardless of whether or not auto-refresh is used.




### <a name="Resolver.Fetch">func</a> (\*Resolver) [Fetch](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=2914:2972#L91)
``` go
func (r *Resolver) Fetch(address string) ([]net.IP, error)
```
Fetch returns a collection of IPs from cache, or a live lookup if not.


##### Example Resolver_Fetch:
``` go
//refresh items every 5 minutes
resolver := New(time.Minute * 5)
//get an array of net.IP
ips, _ := resolver.Fetch("dns.google.com")
fmt.Printf("%+v\n", ips)
```



### <a name="Resolver.FetchOne">func</a> (\*Resolver) [FetchOne](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=3078:3137#L96)
``` go
func (r *Resolver) FetchOne(address string) (net.IP, error)
```
FetchOne returns a single IP from cache, or a live lookup if not.


##### Example Resolver_FetchOne:
``` go
//refresh items every 5 minutes
resolver := New(time.Minute * 5)
//get the first net.IP
ip, _ := resolver.FetchOne("dns.google.com")
fmt.Printf("%+v\n", ip)
```



### <a name="Resolver.FetchOneString">func</a> (\*Resolver) [FetchOneString](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=3337:3402#L105)
``` go
func (r *Resolver) FetchOneString(address string) (string, error)
```
FetchOneString returns a single IP -as a string- from cache, or a live lookup if not.


##### Example Resolver_FetchOneString:
``` go
//refresh items every 5 minutes
resolver := New(time.Minute * 5)
//get the first net.IP as string
ipString, _ := resolver.FetchOneString("dns.google.com")
fmt.Printf("%s\n", ipString)
```



### <a name="Resolver.Lookup">func</a> (\*Resolver) [Lookup](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=4048:4107#L126)
``` go
func (r *Resolver) Lookup(address string) ([]net.IP, error)
```
Lookup returns a collection of IPs from a live lookup, and updates the cache.
Most callers should use one of the Fetch functions.




### <a name="Resolver.Purge">func</a> (\*Resolver) [Purge](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=4209:4235#L131)
``` go
func (r *Resolver) Purge()
```
Purge will remove all entries. To comply with ResolverCache.




### <a name="Resolver.Refresh">func</a> (\*Resolver) [Refresh](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=3614:3642#L114)
``` go
func (r *Resolver) Refresh()
```
Refresh will iterate over cache items, and performing a live lookup one every RefreshSleepTime.




### <a name="Resolver.RefreshTimeout">func</a> (\*Resolver) [RefreshTimeout](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=3824:3880#L120)
``` go
func (r *Resolver) RefreshTimeout(timeout time.Duration)
```
RefreshTimeout will iterate over cache items, and performing a live lookup one every RefreshSleepTime,
until completed or the stated timeout expires.




## <a name="ResolverCache">type</a> [ResolverCache](https://github.com/cognusion/dnscache/tree/master/cache.go?s=168:1467#L10)
``` go
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
```
ResolverCache is an interface to define different caches for Resolver.
All functions defined here must be goro-safe.










## <a name="ResolverConfig">type</a> [ResolverConfig](https://github.com/cognusion/dnscache/tree/master/cache.go?s=1541:1676#L43)
``` go
type ResolverConfig struct {
    Cache               ResolverCache
    AutoRefreshInterval time.Duration
    AutoRefreshTimeout  time.Duration
}

```
ResolverConfig is a common configuration structure for the Resolver.














- - -
Generated by [godoc2md](http://github.com/cognusion/godoc2md)
