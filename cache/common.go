// Package cache provides caching options to DNSCache, or other similar consumers.
// The instantiated caches here must implement `dnscache.ResolverCache`.
package cache

import (
	"errors"
	"fmt"
	"net"
	"slices"
)

var (
	// ErrorConfigKeyUnsupported is returned by cache constructors when a ConfigOption passed is unsupported.
	ErrorConfigKeyUnsupported = errors.New("option is not supported")

	// DefaultResolver is the resolver that will be used if nothing is passed to a constructor.
	DefaultResolver ResolverFunc = net.LookupIP
)

// ResolverFunc is a type to allow abtracting of the lowest resolver logic.
type ResolverFunc func(address string) ([]net.IP, error)

// ConfigKey is a string type for static config key name consistency
type ConfigKey string

// Error is for returning a context-relevant value-type-mismatch error
func (c ConfigKey) Error() error {
	return fmt.Errorf("value of option %s is the wrong type", string(c))
}

// IsIn checks the collection for itself, returning the value and true if it is found,
// or nil and false.
func (c ConfigKey) IsIn(in []ConfigOption) (any, bool) {
	for _, o := range in {
		if o.Key == c {
			return o.Value, true
		}
	}
	return nil, false
}

// ConfigOption is a simple tuple for passing options.
type ConfigOption struct {
	Key   ConfigKey
	Value any
}

// NewConfigOption is a helper function for creating ConfigOptions.
func NewConfigOption(key ConfigKey, value any) ConfigOption {
	return ConfigOption{
		Key:   key,
		Value: value,
	}
}

// ipsTov4 takes a list of net.IPs and returns a []string of those that are valid ipv4s.
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
