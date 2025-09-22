

# cache
`import "github.com/cognusion/dnscache/cache"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)

## <a name="pkg-overview">Overview</a>
Package cache provides caching options to DNSCache, or other similar consumers.
The instantiated caches here must implement `dnscache.ResolverCache`.




## <a name="pkg-index">Index</a>
* [Constants](#pkg-constants)
* [type ConfigKey](#ConfigKey)
  * [func (c ConfigKey) Error() error](#ConfigKey.Error)
  * [func (c ConfigKey) IsIn(in []ConfigOption) (any, bool)](#ConfigKey.IsIn)
* [type ConfigOption](#ConfigOption)
  * [func NewConfigOption(key ConfigKey, value any) ConfigOption](#NewConfigOption)
* [type LRU](#LRU)
  * [func NewLRU(options ...ConfigOption) (*LRU, error)](#NewLRU)
  * [func (r *LRU) Add(key string, value []net.IP)](#LRU.Add)
  * [func (r *LRU) Close() error](#LRU.Close)
  * [func (r *LRU) Fetch(address string) ([]net.IP, error)](#LRU.Fetch)
  * [func (r *LRU) Get(key string) ([]net.IP, bool)](#LRU.Get)
  * [func (r *LRU) Len() int](#LRU.Len)
  * [func (r *LRU) Lookup(address string) ([]net.IP, error)](#LRU.Lookup)
  * [func (r *LRU) Purge()](#LRU.Purge)
  * [func (r *LRU) Refresh(timeout time.Duration)](#LRU.Refresh)
  * [func (r *LRU) Remove(key string)](#LRU.Remove)
* [type ResolverFunc](#ResolverFunc)
* [type Simple](#Simple)
  * [func NewSimple(options ...ConfigOption) (*Simple, error)](#NewSimple)
  * [func (r *Simple) Add(address string, ips []net.IP)](#Simple.Add)
  * [func (r *Simple) Close() error](#Simple.Close)
  * [func (r *Simple) Fetch(address string) ([]net.IP, error)](#Simple.Fetch)
  * [func (r *Simple) Get(address string) ([]net.IP, bool)](#Simple.Get)
  * [func (r *Simple) Len() int](#Simple.Len)
  * [func (r *Simple) Lookup(address string) ([]net.IP, error)](#Simple.Lookup)
  * [func (r *Simple) Purge()](#Simple.Purge)
  * [func (r *Simple) Refresh(timeout time.Duration)](#Simple.Refresh)
  * [func (r *Simple) Remove(address string)](#Simple.Remove)


#### <a name="pkg-files">Package files</a>
[common.go](https://github.com/cognusion/dnscache/tree/master/cache/common.go) [lru.go](https://github.com/cognusion/dnscache/tree/master/cache/lru.go) [map.go](https://github.com/cognusion/dnscache/tree/master/cache/map.go)


## <a name="pkg-constants">Constants</a>
``` go
const (
    // ConfigSize is an int.
    // Values > 0 represent the number of items allowed in the cache.
    // 0 is unlimited.
    ConfigSize = ConfigKey("CacheSize")
    // ConfigItemTTL is a time.Duration.
    // Values will control the life of unaccessed items in the cache.
    ConfigItemTTL = ConfigKey("ItemTTL")
    // ConfigAllowRefresh is a bool.
    // True allows the cache to perform Refresh operations.
    // False requires the cache to silently decline Refresh operations.
    ConfigAllowRefresh = ConfigKey("AllowRefresh")
)
```
``` go
const (
    // ConfigResolver is a ResolverFunc.
    ConfigResolver = ConfigKey("Resolver")
    // ConfigRefreshShuffle is a bool.
    // True will "shuffle" cache items before the Refresh begins.
    ConfigRefreshShuffle = ConfigKey("RefreshShuffle")
    // ConfigRefreshSleepTime is a time.Duration.
    // If > 0 then a Refresh pass, if any, will wait that Duration between item lookups.
    // 0 disables the delay.
    ConfigRefreshSleepTime = ConfigKey("RefreshSleepTime")
)
```




## <a name="ConfigKey">type</a> [ConfigKey](https://github.com/cognusion/dnscache/tree/master/cache/common.go?s=732:753#L23)
``` go
type ConfigKey string
```
ConfigKey is a string type for static config key name consistency










### <a name="ConfigKey.Error">func</a> (ConfigKey) [Error](https://github.com/cognusion/dnscache/tree/master/cache/common.go?s=826:858#L26)
``` go
func (c ConfigKey) Error() error
```
Error is for returning a context-relevant value-type-mismatch error




### <a name="ConfigKey.IsIn">func</a> (ConfigKey) [IsIn](https://github.com/cognusion/dnscache/tree/master/cache/common.go?s=1042:1096#L32)
``` go
func (c ConfigKey) IsIn(in []ConfigOption) (any, bool)
```
IsIn checks the collection for itself, returning the value and true if it is found,
or nil and false.




## <a name="ConfigOption">type</a> [ConfigOption](https://github.com/cognusion/dnscache/tree/master/cache/common.go?s=1249:1305#L42)
``` go
type ConfigOption struct {
    Key   ConfigKey
    Value any
}

```
ConfigOption is a simple tuple for passing options.







### <a name="NewConfigOption">func</a> [NewConfigOption](https://github.com/cognusion/dnscache/tree/master/cache/common.go?s=1375:1434#L48)
``` go
func NewConfigOption(key ConfigKey, value any) ConfigOption
```
NewConfigOption is a helper function for creating ConfigOptions.





## <a name="LRU">type</a> [LRU](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=1525:1670#L55)
``` go
type LRU struct {
    // contains filtered or unexported fields
}

```
LRU is a "least recently used" cache of fixed size, that evicts items
when necessary to free space for more. If ItemTTL is specified, then
the cache will automatically evict items that are unaccessed beyond that point.







### <a name="NewLRU">func</a> [NewLRU](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=2036:2086#L69)
``` go
func NewLRU(options ...ConfigOption) (*LRU, error)
```
NewLRU instantiates an LRU cache.
If ItemTTL is specified, an expirable cache is created, otherwise a twoqueue cache is used.
Valid ConfigOptions are: Resolver, RefreshShuffle, RefreshSleepTime, AllowRefresh, ItemTTL, Size.
Required are: Size.
Defaults are: Resolver(DefaultResolver), RefreshShuffle(true), RefreshSleepTime(1s), AllowRefresh(true).





### <a name="LRU.Add">func</a> (\*LRU) [Add](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=6232:6277#L256)
``` go
func (r *LRU) Add(key string, value []net.IP)
```
Add will upsert a collection into the cache.




### <a name="LRU.Close">func</a> (\*LRU) [Close](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=6139:6166#L251)
``` go
func (r *LRU) Close() error
```
Close is a noop. Satisfies ResolverCache




### <a name="LRU.Fetch">func</a> (\*LRU) [Fetch](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=4091:4144#L158)
``` go
func (r *LRU) Fetch(address string) ([]net.IP, error)
```
Fetch retrieves a collection from the cache,
or performs a live lookup and adds it to the cache.




### <a name="LRU.Get">func</a> (\*LRU) [Get](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=6524:6570#L267)
``` go
func (r *LRU) Get(key string) ([]net.IP, bool)
```
Get will return a collection from the cache, also bool if
a collection was retrieved.




### <a name="LRU.Len">func</a> (\*LRU) [Len](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=6654:6677#L272)
``` go
func (r *LRU) Len() int
```
Len will return the number of items in the cache.




### <a name="LRU.Lookup">func</a> (\*LRU) [Lookup](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=4320:4374#L169)
``` go
func (r *LRU) Lookup(address string) ([]net.IP, error)
```
Lookup performs a live lookup,
and adds the results to the cache.




### <a name="LRU.Purge">func</a> (\*LRU) [Purge](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=4541:4562#L180)
``` go
func (r *LRU) Purge()
```
Purge removes all entries from the cache.




### <a name="LRU.Refresh">func</a> (\*LRU) [Refresh](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=4654:4698#L185)
``` go
func (r *LRU) Refresh(timeout time.Duration)
```
Refresh will crawl the keys and update the cache with new values.




### <a name="LRU.Remove">func</a> (\*LRU) [Remove](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=6373:6405#L261)
``` go
func (r *LRU) Remove(key string)
```
Remove will remove a collection from the cache, if it exists.




## <a name="ResolverFunc">type</a> [ResolverFunc](https://github.com/cognusion/dnscache/tree/master/cache/common.go?s=605:661#L20)
``` go
type ResolverFunc func(address string) ([]net.IP, error)
```
ResolverFunc is a type to allow abtracting of the lowest resolver logic.


``` go
var (
    // ErrorConfigKeyUnsupported is returned by cache constructors when a ConfigOption passed is unsupported.
    ErrorConfigKeyUnsupported = errors.New("option is not supported")

    // DefaultResolver is the resolver that will be used if nothing is passed to a constructor.
    DefaultResolver ResolverFunc = net.LookupIP
)
```









## <a name="Simple">type</a> [Simple](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=607:784#L26)
``` go
type Simple struct {
    // contains filtered or unexported fields
}

```
Simple is a mutex-controlled map-based ResolverCache.







### <a name="NewSimple">func</a> [NewSimple](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=1010:1066#L40)
``` go
func NewSimple(options ...ConfigOption) (*Simple, error)
```
NewSimple instantiates a Simple cache.
Valid ConfigOptions are: Resolver, RefreshShuffle, RefreshSleepTime.
Required are: none.
Defaults are: Resolver(DefaultResolver), RefreshShuffle(true), RefreshSleepTime(1s)





### <a name="Simple.Add">func</a> (\*Simple) [Add](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=4246:4296#L184)
``` go
func (r *Simple) Add(address string, ips []net.IP)
```
Add will upsert a collection into the cache.




### <a name="Simple.Close">func</a> (\*Simple) [Close](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=4135:4165#L178)
``` go
func (r *Simple) Close() error
```
Close will signal an in-progress Refresh, if any, to exit.




### <a name="Simple.Fetch">func</a> (\*Simple) [Fetch](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=2115:2171#L90)
``` go
func (r *Simple) Fetch(address string) ([]net.IP, error)
```
Fetch retrieves a collection from the cache,
or performs a live lookup and adds it to the cache.




### <a name="Simple.Get">func</a> (\*Simple) [Get](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=4618:4671#L199)
``` go
func (r *Simple) Get(address string) ([]net.IP, bool)
```
Get will return a collection from the cache, also bool if
a collection was retrieved.




### <a name="Simple.Len">func</a> (\*Simple) [Len](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=4806:4832#L208)
``` go
func (r *Simple) Len() int
```
Len will return the number of items in the cache.




### <a name="Simple.Lookup">func</a> (\*Simple) [Lookup](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=2441:2498#L103)
``` go
func (r *Simple) Lookup(address string) ([]net.IP, error)
```
Lookup returns a collection of IPs from a live lookup, and updates the cache.
Most callers should use one of the Fetch functions.




### <a name="Simple.Purge">func</a> (\*Simple) [Purge](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=2694:2718#L116)
``` go
func (r *Simple) Purge()
```
Purge removes all entries from the cache.




### <a name="Simple.Refresh">func</a> (\*Simple) [Refresh](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=2988:3035#L126)
``` go
func (r *Simple) Refresh(timeout time.Duration)
```
Refresh will crawl the cache and update their entries.
A timeout of 0 must mean no timeout.
RefreshSleepTime is checked for per-lookup intervals.
RefreshShuffle is checked.




### <a name="Simple.Remove">func</a> (\*Simple) [Remove](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=4423:4462#L191)
``` go
func (r *Simple) Remove(address string)
```
Remove will remove a collection from the cache, if it exists.








- - -
Generated by [godoc2md](http://github.com/cognusion/godoc2md)
