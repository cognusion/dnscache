

# dnscache
`import "github.com/cognusion/dnscache"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)
* [Examples](#pkg-examples)

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
  * [func NewWithRefreshTimeout(refreshRate, refreshTimeout time.Duration) *Resolver](#NewWithRefreshTimeout)
  * [func (r *Resolver) Close() error](#Resolver.Close)
  * [func (r *Resolver) Fetch(address string) ([]net.IP, error)](#Resolver.Fetch)
  * [func (r *Resolver) FetchOne(address string) (net.IP, error)](#Resolver.FetchOne)
  * [func (r *Resolver) FetchOneString(address string) (string, error)](#Resolver.FetchOneString)
  * [func (r *Resolver) Lookup(address string) ([]net.IP, error)](#Resolver.Lookup)
  * [func (r *Resolver) Refresh()](#Resolver.Refresh)
  * [func (r *Resolver) RefreshTimeout(timeout time.Duration)](#Resolver.RefreshTimeout)

#### <a name="pkg-examples">Examples</a>
* [Package](#example-)
* [New](#example-new)
* [NewWithRefreshTimeout](#example-newwithrefreshtimeout)
* [Resolver.Fetch](#example-resolver_fetch)
* [Resolver.FetchOne](#example-resolver_fetchone)
* [Resolver.FetchOneString](#example-resolver_fetchonestring)

#### <a name="pkg-files">Package files</a>
[dnscache.go](https://github.com/cognusion/dnscache/tree/master/dnscache.go)



## <a name="pkg-variables">Variables</a>
``` go
var RefreshSleepTime = 1 * time.Second
```
RefreshSleepTime is the delay between Refresh (and auto-refresh)
lookups, to keep the resolver threads from piling up.




## <a name="Resolver">type</a> [Resolver](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=569:661#L23)
``` go
type Resolver struct {
    // contains filtered or unexported fields
}

```
Resolver is a goro-safe caching DNS resolver.







### <a name="New">func</a> [New](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=810:855#L32)
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

### <a name="NewWithRefreshTimeout">func</a> [NewWithRefreshTimeout](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=1207:1286#L41)
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




### <a name="Resolver.Close">func</a> (\*Resolver) [Close](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=1646:1678#L54)
``` go
func (r *Resolver) Close() error
```
Close signals the auto-refresh goro, if any, to quit.
This is safe to call once, in any thread, regardless of whether or not auto-refresh is used.




### <a name="Resolver.Fetch">func</a> (\*Resolver) [Fetch](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=1785:1843#L60)
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



### <a name="Resolver.FetchOne">func</a> (\*Resolver) [FetchOne](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=2046:2105#L72)
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



### <a name="Resolver.FetchOneString">func</a> (\*Resolver) [FetchOneString](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=2305:2370#L81)
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



### <a name="Resolver.Lookup">func</a> (\*Resolver) [Lookup](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=3899:3958#L156)
``` go
func (r *Resolver) Lookup(address string) ([]net.IP, error)
```
Lookup returns a collection of IPs from a live lookup, and updates the cache.




### <a name="Resolver.Refresh">func</a> (\*Resolver) [Refresh](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=2582:2610#L90)
``` go
func (r *Resolver) Refresh()
```
Refresh will iterate over cache items, and performing a live lookup one every RefreshSleepTime.




### <a name="Resolver.RefreshTimeout">func</a> (\*Resolver) [RefreshTimeout](https://github.com/cognusion/dnscache/tree/master/dnscache.go?s=2793:2849#L96)
``` go
func (r *Resolver) RefreshTimeout(timeout time.Duration)
```
RefreshTimeout will iterate over cache items, and performing a live lookup one every RefreshSleepTime,
until completed or the stated timeout expires.








- - -
Generated by [godoc2md](http://github.com/cognusion/godoc2md)
