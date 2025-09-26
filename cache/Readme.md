

# cache
`import "github.com/cognusion/dnscache/cache"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)

## <a name="pkg-overview">Overview</a>
Package cache provides caching options to DNSCache, or other similar consumers.
The instantiated caches here must implement `dnscache.ResolverCache`.




## <a name="pkg-index">Index</a>
* [Constants](#pkg-constants)
* [func BatchRefresh(cache RefreshableCache, resolver ResolverFunc, options ...ConfigOption) (bool, error)](#BatchRefresh)
* [func LinearRefresh(cache RefreshableCache, resolver ResolverFunc, options ...ConfigOption) (bool, error)](#LinearRefresh)
* [func NoRefresh(cache RefreshableCache, resolver ResolverFunc, options ...ConfigOption) (bool, error)](#NoRefresh)
* [type ConfigKey](#ConfigKey)
  * [func (c ConfigKey) Error() error](#ConfigKey.Error)
  * [func (c ConfigKey) IsIn(in []ConfigOption) (any, bool)](#ConfigKey.IsIn)
* [type ConfigOption](#ConfigOption)
  * [func NewConfigOption(key ConfigKey, value any) ConfigOption](#NewConfigOption)
* [type LRU](#LRU)
  * [func NewLRU(options ...ConfigOption) (*LRU, error)](#NewLRU)
  * [func (r *LRU) Add(key string, value []net.IP)](#LRU.Add)
  * [func (r *LRU) Close() error](#LRU.Close)
  * [func (r *LRU) Contains(address string) bool](#LRU.Contains)
  * [func (r *LRU) Fetch(address string) ([]net.IP, error)](#LRU.Fetch)
  * [func (r *LRU) Get(key string) ([]net.IP, bool)](#LRU.Get)
  * [func (r *LRU) Keys() []string](#LRU.Keys)
  * [func (r *LRU) Len() int](#LRU.Len)
  * [func (r *LRU) Lookup(address string) ([]net.IP, error)](#LRU.Lookup)
  * [func (r *LRU) Purge()](#LRU.Purge)
  * [func (r *LRU) Refresh(timeout time.Duration)](#LRU.Refresh)
  * [func (r *LRU) Remove(key string)](#LRU.Remove)
* [type RefreshFunc](#RefreshFunc)
* [type RefreshType](#RefreshType)
* [type RefreshableCache](#RefreshableCache)
* [type ResolverFunc](#ResolverFunc)
* [type Simple](#Simple)
  * [func NewSimple(options ...ConfigOption) (*Simple, error)](#NewSimple)
  * [func (r *Simple) Add(address string, ips []net.IP)](#Simple.Add)
  * [func (r *Simple) Close() error](#Simple.Close)
  * [func (r *Simple) Contains(address string) bool](#Simple.Contains)
  * [func (r *Simple) Fetch(address string) ([]net.IP, error)](#Simple.Fetch)
  * [func (r *Simple) Get(address string) ([]net.IP, bool)](#Simple.Get)
  * [func (r *Simple) Keys() []string](#Simple.Keys)
  * [func (r *Simple) Len() int](#Simple.Len)
  * [func (r *Simple) Lookup(address string) ([]net.IP, error)](#Simple.Lookup)
  * [func (r *Simple) Purge()](#Simple.Purge)
  * [func (r *Simple) Refresh(timeout time.Duration)](#Simple.Refresh)
  * [func (r *Simple) Remove(address string)](#Simple.Remove)


#### <a name="pkg-files">Package files</a>
[common.go](https://github.com/cognusion/dnscache/tree/master/cache/common.go) [lru.go](https://github.com/cognusion/dnscache/tree/master/cache/lru.go) [map.go](https://github.com/cognusion/dnscache/tree/master/cache/map.go) [refresh.go](https://github.com/cognusion/dnscache/tree/master/cache/refresh.go)


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
``` go
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
```



## <a name="BatchRefresh">func</a> [BatchRefresh](https://github.com/cognusion/dnscache/tree/master/cache/refresh.go?s=4033:4136#L137)
``` go
func BatchRefresh(cache RefreshableCache, resolver ResolverFunc, options ...ConfigOption) (bool, error)
```
BatchRefresh uses workers to do RefreshBatchSize lookups at a time. By default, it will shuffle the keys,
sleep 1s between each batch, and run until it is done (no timeout).



## <a name="LinearRefresh">func</a> [LinearRefresh](https://github.com/cognusion/dnscache/tree/master/cache/refresh.go?s=1652:1756#L44)
``` go
func LinearRefresh(cache RefreshableCache, resolver ResolverFunc, options ...ConfigOption) (bool, error)
```
LinearRefresh is the classic ordered, one-at-a-time RefreshFunc. By default, it will shuffle the keys,
sleep for 1s between each lookup, and continue until it is done (no timeout).



## <a name="NoRefresh">func</a> [NoRefresh](https://github.com/cognusion/dnscache/tree/master/cache/refresh.go?s=1341:1441#L38)
``` go
func NoRefresh(cache RefreshableCache, resolver ResolverFunc, options ...ConfigOption) (bool, error)
```
NoRefresh is a noop RefreshFunc that always returns true, and never an error.




## <a name="ConfigKey">type</a> [ConfigKey](https://github.com/cognusion/dnscache/tree/master/cache/common.go?s=1236:1257#L38)
``` go
type ConfigKey string
```
ConfigKey is a string type for static config key name consistency










### <a name="ConfigKey.Error">func</a> (ConfigKey) [Error](https://github.com/cognusion/dnscache/tree/master/cache/common.go?s=1330:1362#L41)
``` go
func (c ConfigKey) Error() error
```
Error is for returning a context-relevant value-type-mismatch error




### <a name="ConfigKey.IsIn">func</a> (ConfigKey) [IsIn](https://github.com/cognusion/dnscache/tree/master/cache/common.go?s=1546:1600#L47)
``` go
func (c ConfigKey) IsIn(in []ConfigOption) (any, bool)
```
IsIn checks the collection for itself, returning the value and true if it is found,
or nil and false.




## <a name="ConfigOption">type</a> [ConfigOption](https://github.com/cognusion/dnscache/tree/master/cache/common.go?s=1753:1809#L57)
``` go
type ConfigOption struct {
    Key   ConfigKey
    Value any
}

```
ConfigOption is a simple tuple for passing options.







### <a name="NewConfigOption">func</a> [NewConfigOption](https://github.com/cognusion/dnscache/tree/master/cache/common.go?s=1879:1938#L63)
``` go
func NewConfigOption(key ConfigKey, value any) ConfigOption
```
NewConfigOption is a helper function for creating ConfigOptions.





## <a name="LRU">type</a> [LRU](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=1290:1494#L49)
``` go
type LRU struct {
    // contains filtered or unexported fields
}

```
LRU is a "least recently used" cache of fixed size, that evicts items
when necessary to free space for more. If ItemTTL is specified, then
the cache will automatically evict items that are unaccessed beyond that point.







### <a name="NewLRU">func</a> [NewLRU](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=1860:1910#L65)
``` go
func NewLRU(options ...ConfigOption) (*LRU, error)
```
NewLRU instantiates an LRU cache.
If ItemTTL is specified, an expirable cache is created, otherwise a twoqueue cache is used.
Valid ConfigOptions are: Resolver, RefreshShuffle, RefreshSleepTime, AllowRefresh, ItemTTL, Size.
Required are: Size.
Defaults are: Resolver(DefaultResolver), RefreshShuffle(true), RefreshSleepTime(1s), AllowRefresh(true).





### <a name="LRU.Add">func</a> (\*LRU) [Add](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=5876:5921#L234)
``` go
func (r *LRU) Add(key string, value []net.IP)
```
Add will upsert a collection into the cache.




### <a name="LRU.Close">func</a> (\*LRU) [Close](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=5783:5810#L229)
``` go
func (r *LRU) Close() error
```
Close is a noop. Satisfies ResolverCache




### <a name="LRU.Contains">func</a> (\*LRU) [Contains](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=6402:6445#L255)
``` go
func (r *LRU) Contains(address string) bool
```
Contains returns true if a value is in the cache.




### <a name="LRU.Fetch">func</a> (\*LRU) [Fetch](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=4482:4535#L177)
``` go
func (r *LRU) Fetch(address string) ([]net.IP, error)
```
Fetch retrieves a collection from the cache,
or performs a live lookup and adds it to the cache.




### <a name="LRU.Get">func</a> (\*LRU) [Get](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=6168:6214#L245)
``` go
func (r *LRU) Get(key string) ([]net.IP, bool)
```
Get will return a collection from the cache, also bool if
a collection was retrieved.




### <a name="LRU.Keys">func</a> (\*LRU) [Keys](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=6534:6563#L260)
``` go
func (r *LRU) Keys() []string
```
Keys returns a sorted slice of the cache keys




### <a name="LRU.Len">func</a> (\*LRU) [Len](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=6298:6321#L250)
``` go
func (r *LRU) Len() int
```
Len will return the number of items in the cache.




### <a name="LRU.Lookup">func</a> (\*LRU) [Lookup](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=4711:4765#L188)
``` go
func (r *LRU) Lookup(address string) ([]net.IP, error)
```
Lookup performs a live lookup,
and adds the results to the cache.




### <a name="LRU.Purge">func</a> (\*LRU) [Purge](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=4932:4953#L199)
``` go
func (r *LRU) Purge()
```
Purge removes all entries from the cache.




### <a name="LRU.Refresh">func</a> (\*LRU) [Refresh](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=5045:5089#L204)
``` go
func (r *LRU) Refresh(timeout time.Duration)
```
Refresh will crawl the keys and update the cache with new values.




### <a name="LRU.Remove">func</a> (\*LRU) [Remove](https://github.com/cognusion/dnscache/tree/master/cache/lru.go?s=6017:6049#L239)
``` go
func (r *LRU) Remove(key string)
```
Remove will remove a collection from the cache, if it exists.




## <a name="RefreshFunc">type</a> [RefreshFunc](https://github.com/cognusion/dnscache/tree/master/cache/common.go?s=947:1031#L32)
``` go
type RefreshFunc func(RefreshableCache, ResolverFunc, ...ConfigOption) (bool, error)
```
RefreshFunc is a definition for a Refreshable Refresh. How refreshing!
Do you feel refreshed? How many more times will I say "refresh"?
Refresh.










## <a name="RefreshType">type</a> [RefreshType](https://github.com/cognusion/dnscache/tree/master/cache/common.go?s=594:617#L21)
``` go
type RefreshType string
```
RefreshType is a string type for static consistency










## <a name="RefreshableCache">type</a> [RefreshableCache](https://github.com/cognusion/dnscache/tree/master/cache/common.go?s=708:791#L24)
``` go
type RefreshableCache interface {
    Keys() []string
    Contains(address string) bool
}
```
RefreshableCache is a minimal interface that caches must implement to be Refreshable.










## <a name="ResolverFunc">type</a> [ResolverFunc](https://github.com/cognusion/dnscache/tree/master/cache/common.go?s=1109:1165#L35)
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









## <a name="Simple">type</a> [Simple](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=587:846#L25)
``` go
type Simple struct {
    // contains filtered or unexported fields
}

```
Simple is a mutex-controlled map-based ResolverCache.







### <a name="NewSimple">func</a> [NewSimple](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=1072:1128#L42)
``` go
func NewSimple(options ...ConfigOption) (*Simple, error)
```
NewSimple instantiates a Simple cache.
Valid ConfigOptions are: Resolver, RefreshShuffle, RefreshSleepTime.
Required are: none.
Defaults are: Resolver(DefaultResolver), RefreshShuffle(true), RefreshSleepTime(1s)





### <a name="Simple.Add">func</a> (\*Simple) [Add](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=4447:4497#L183)
``` go
func (r *Simple) Add(address string, ips []net.IP)
```
Add will upsert a collection into the cache.




### <a name="Simple.Close">func</a> (\*Simple) [Close](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=4336:4366#L177)
``` go
func (r *Simple) Close() error
```
Close will signal an in-progress Refresh, if any, to exit.




### <a name="Simple.Contains">func</a> (\*Simple) [Contains](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=5153:5199#L214)
``` go
func (r *Simple) Contains(address string) bool
```
Contains returns true if a value is in the cache.




### <a name="Simple.Fetch">func</a> (\*Simple) [Fetch](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=2704:2760#L116)
``` go
func (r *Simple) Fetch(address string) ([]net.IP, error)
```
Fetch retrieves a collection from the cache,
or performs a live lookup and adds it to the cache.




### <a name="Simple.Get">func</a> (\*Simple) [Get](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=4819:4872#L198)
``` go
func (r *Simple) Get(address string) ([]net.IP, bool)
```
Get will return a collection from the cache, also bool if
a collection was retrieved.




### <a name="Simple.Keys">func</a> (\*Simple) [Keys](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=5327:5359#L223)
``` go
func (r *Simple) Keys() []string
```
Keys returns a sorted slice of the cache keys




### <a name="Simple.Len">func</a> (\*Simple) [Len](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=5007:5033#L207)
``` go
func (r *Simple) Len() int
```
Len will return the number of items in the cache.




### <a name="Simple.Lookup">func</a> (\*Simple) [Lookup](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=3030:3087#L129)
``` go
func (r *Simple) Lookup(address string) ([]net.IP, error)
```
Lookup returns a collection of IPs from a live lookup, and updates the cache.
Most callers should use one of the Fetch functions.




### <a name="Simple.Purge">func</a> (\*Simple) [Purge](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=3283:3307#L142)
``` go
func (r *Simple) Purge()
```
Purge removes all entries from the cache.




### <a name="Simple.Refresh">func</a> (\*Simple) [Refresh](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=3577:3624#L152)
``` go
func (r *Simple) Refresh(timeout time.Duration)
```
Refresh will crawl the cache and update their entries.
A timeout of 0 must mean no timeout.
RefreshSleepTime is checked for per-lookup intervals.
RefreshShuffle is checked.




### <a name="Simple.Remove">func</a> (\*Simple) [Remove](https://github.com/cognusion/dnscache/tree/master/cache/map.go?s=4624:4663#L190)
``` go
func (r *Simple) Remove(address string)
```
Remove will remove a collection from the cache, if it exists.








- - -
Generated by [godoc2md](http://github.com/cognusion/godoc2md)
