package cache

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

type Item struct {
	Object          interface{}
	Expiration      int64
	RefreshDeadline int64
}

// Returns true if the item has expired.
func (item Item) Expired() bool {
	if item.Expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > item.Expiration
}

// Returns true if the item has reached its refresh deadline.
func (item Item) RefreshDeadlineReached() bool {
	if item.RefreshDeadline == 0 {
		return false
	}
	return time.Now().UnixNano() > item.RefreshDeadline
}

const (
	// For use with functions that take an expiration time.
	NoExpiration time.Duration = -1
	// For use with functions that take an expiration time.
	NoRefreshDeadline time.Duration = -1
	// For use with functions that take an expiration time. Equivalent to
	// passing in the same expiration duration as was given to New() or
	// NewFrom() when the cache was created (e.g. 5 minutes.)
	DefaultExpiration time.Duration = 0
)

type Cache struct {
	*cache
	// If this is confusing, see the comment at the bottom of New()
}

type cache struct {
	defaultExpiration       time.Duration
	storage                 Storage
	onRefreshNeeded         func(string)
	refreshConcurrencyMap   map[string]bool
	refreshConcurrencyMutex sync.Mutex
	refreshKeys             chan string
}

// Add an item to the cache, replacing any existing item. If the duration is 0
// (DefaultExpiration), the cache's default expiration time is used. If it is -1
// (NoExpiration), the item never expires.
func (c *cache) Set(k string, x interface{}, d time.Duration, rd time.Duration) {
	// "Inlining" of set
	var e int64
	var erd int64
	if d == DefaultExpiration {
		d = c.defaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	if rd > 0 {
		erd = time.Now().Add(rd).UnixNano()
	}
	c.storage.Lock()
	item := Item{
		Object:     x,
		Expiration: e,
		RefreshDeadline: erd,
	}
	c.storage.Set(k, item)
	// TODO: Calls to mu.Unlock are currently not deferred because defer
	// adds ~200 ns (as of go1.)
	c.storage.Unlock()
}

func (c *cache) set(k string, x interface{}, d time.Duration, rd time.Duration) {
	var e int64
	var erd int64
	if d == DefaultExpiration {
		d = c.defaultExpiration
	}
	if d > 0 {
		e = time.Now().Add(d).UnixNano()
	}
	if rd > 0 {
		erd = time.Now().Add(rd).UnixNano()
	}
	item := Item{
		Object:     x,
		Expiration: e,
		RefreshDeadline: erd,
	}
	c.storage.Set(k, item)

}

// Add an item to the cache only if an item doesn't already exist for the given
// key, or if the existing item has expired. Returns an error otherwise.
func (c *cache) Add(k string, x interface{}, d time.Duration, rd time.Duration) error {
	c.storage.Lock()
	_, found := c.get(k)
	if found {
		c.storage.Unlock()
		return fmt.Errorf("Item %s already exists", k)
	}
	c.set(k, x, d, rd)
	c.storage.Unlock()
	return nil
}

// Set a new value for the cache key only if it already exists, and the existing
// item hasn't expired. Returns an error otherwise.
func (c *cache) Replace(k string, x interface{}, d time.Duration, rd time.Duration) error {
	c.storage.Lock()
	_, found := c.get(k)
	if !found {
		c.storage.Unlock()
		return fmt.Errorf("Item %s doesn't exist", k)
	}
	c.set(k, x, d, rd)
	c.storage.Unlock()
	return nil
}

func (c *cache) refreshWorker(id int, jobs <-chan string) {
	for k := range jobs {
		c.onRefreshNeeded(k)
		c.refreshConcurrencyMutex.Lock()
		delete(c.refreshConcurrencyMap, k)
		c.refreshConcurrencyMutex.Unlock()

	}
}

// Get an item from the cache. Returns the item or nil, and a bool indicating
// whether the key was found.
func (c *cache) Get(k string) (interface{}, bool) {
	c.storage.RLock()
	// "Inlining" of get and Expired
	item, found := c.storage.Get(k)
	if !found {
		c.storage.RUnlock()
		return nil, false
	}
	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			c.storage.RUnlock()
			return nil, false
		}
	}
	if item.RefreshDeadline > 0 {
		if item.RefreshDeadlineReached() {
			c.refreshConcurrencyMutex.Lock()
			if _, ok := c.refreshConcurrencyMap[k]; !ok {
				c.refreshConcurrencyMap[k] = true
				c.refreshConcurrencyMutex.Unlock()
				go func() {
					c.refreshKeys <- k
				}()
			} else {
				c.refreshConcurrencyMutex.Unlock()
			}

		}
	}
	c.storage.RUnlock()
	return item.Object, true
}

func (c *cache) get(k string) (interface{}, bool) {
	item, found := c.storage.Get(k)
	if !found {
		return nil, false
	}
	// "Inlining" of Expired
	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			return nil, false
		}
	}
	if item.RefreshDeadline > 0 {
		if item.RefreshDeadlineReached() {
			c.refreshConcurrencyMutex.Lock()
			if _, ok := c.refreshConcurrencyMap[k]; !ok {
				c.refreshConcurrencyMap[k] = true
				c.refreshConcurrencyMutex.Unlock()
				go func() {
					c.refreshKeys <- k
				}()
				c.refreshConcurrencyMutex.Lock()
				delete(c.refreshConcurrencyMap, k)
				c.refreshConcurrencyMutex.Unlock()
			} else {
				c.refreshConcurrencyMutex.Unlock()
			}

		}
	}
	return item.Object, true
}

// Increment an item of type int, int8, int16, int32, int64, uintptr, uint,
// uint8, uint32, or uint64, float32 or float64 by n. Returns an error if the
// item's value is not an integer, if it was not found, or if it is not
// possible to increment it by n. To retrieve the incremented value, use one
// of the specialized methods, e.g. IncrementInt64.
func (c *cache) Increment(k string, n int64) error {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return fmt.Errorf("Item %s not found", k)
	}
	switch v.Object.(type) {
	case int:
		v.Object = v.Object.(int) + int(n)
	case int8:
		v.Object = v.Object.(int8) + int8(n)
	case int16:
		v.Object = v.Object.(int16) + int16(n)
	case int32:
		v.Object = v.Object.(int32) + int32(n)
	case int64:
		v.Object = v.Object.(int64) + n
	case uint:
		v.Object = v.Object.(uint) + uint(n)
	case uintptr:
		v.Object = v.Object.(uintptr) + uintptr(n)
	case uint8:
		v.Object = v.Object.(uint8) + uint8(n)
	case uint16:
		v.Object = v.Object.(uint16) + uint16(n)
	case uint32:
		v.Object = v.Object.(uint32) + uint32(n)
	case uint64:
		v.Object = v.Object.(uint64) + uint64(n)
	case float32:
		v.Object = v.Object.(float32) + float32(n)
	case float64:
		v.Object = v.Object.(float64) + float64(n)
	default:
		c.storage.Unlock()
		return fmt.Errorf("The value for %s is not an integer", k)
	}
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nil
}

// Increment an item of type float32 or float64 by n. Returns an error if the
// item's value is not floating point, if it was not found, or if it is not
// possible to increment it by n. Pass a negative number to decrement the
// value. To retrieve the incremented value, use one of the specialized methods,
// e.g. IncrementFloat64.
func (c *cache) IncrementFloat(k string, n float64) error {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return fmt.Errorf("Item %s not found", k)
	}
	switch v.Object.(type) {
	case float32:
		v.Object = v.Object.(float32) + float32(n)
	case float64:
		v.Object = v.Object.(float64) + n
	default:
		c.storage.Unlock()
		return fmt.Errorf("The value for %s does not have type float32 or float64", k)
	}
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nil
}

// Increment an item of type int by n. Returns an error if the item's value is
// not an int, or if it was not found. If there is no error, the incremented
// value is returned.
func (c *cache) IncrementInt(k string, n int) (int, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int", k)
	}
	nv := rv + n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Increment an item of type int8 by n. Returns an error if the item's value is
// not an int8, or if it was not found. If there is no error, the incremented
// value is returned.
func (c *cache) IncrementInt8(k string, n int8) (int8, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int8)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int8", k)
	}
	nv := rv + n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Increment an item of type int16 by n. Returns an error if the item's value is
// not an int16, or if it was not found. If there is no error, the incremented
// value is returned.
func (c *cache) IncrementInt16(k string, n int16) (int16, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int16)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int16", k)
	}
	nv := rv + n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Increment an item of type int32 by n. Returns an error if the item's value is
// not an int32, or if it was not found. If there is no error, the incremented
// value is returned.
func (c *cache) IncrementInt32(k string, n int32) (int32, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int32)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int32", k)
	}
	nv := rv + n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Increment an item of type int64 by n. Returns an error if the item's value is
// not an int64, or if it was not found. If there is no error, the incremented
// value is returned.
func (c *cache) IncrementInt64(k string, n int64) (int64, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int64)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int64", k)
	}
	nv := rv + n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Increment an item of type uint by n. Returns an error if the item's value is
// not an uint, or if it was not found. If there is no error, the incremented
// value is returned.
func (c *cache) IncrementUint(k string, n uint) (uint, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint", k)
	}
	nv := rv + n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Increment an item of type uintptr by n. Returns an error if the item's value
// is not an uintptr, or if it was not found. If there is no error, the
// incremented value is returned.
func (c *cache) IncrementUintptr(k string, n uintptr) (uintptr, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uintptr)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uintptr", k)
	}
	nv := rv + n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Increment an item of type uint8 by n. Returns an error if the item's value
// is not an uint8, or if it was not found. If there is no error, the
// incremented value is returned.
func (c *cache) IncrementUint8(k string, n uint8) (uint8, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint8)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint8", k)
	}
	nv := rv + n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Increment an item of type uint16 by n. Returns an error if the item's value
// is not an uint16, or if it was not found. If there is no error, the
// incremented value is returned.
func (c *cache) IncrementUint16(k string, n uint16) (uint16, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint16)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint16", k)
	}
	nv := rv + n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Increment an item of type uint32 by n. Returns an error if the item's value
// is not an uint32, or if it was not found. If there is no error, the
// incremented value is returned.
func (c *cache) IncrementUint32(k string, n uint32) (uint32, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint32)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint32", k)
	}
	nv := rv + n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Increment an item of type uint64 by n. Returns an error if the item's value
// is not an uint64, or if it was not found. If there is no error, the
// incremented value is returned.
func (c *cache) IncrementUint64(k string, n uint64) (uint64, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint64)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint64", k)
	}
	nv := rv + n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Increment an item of type float32 by n. Returns an error if the item's value
// is not an float32, or if it was not found. If there is no error, the
// incremented value is returned.
func (c *cache) IncrementFloat32(k string, n float32) (float32, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(float32)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an float32", k)
	}
	nv := rv + n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Increment an item of type float64 by n. Returns an error if the item's value
// is not an float64, or if it was not found. If there is no error, the
// incremented value is returned.
func (c *cache) IncrementFloat64(k string, n float64) (float64, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(float64)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an float64", k)
	}
	nv := rv + n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Decrement an item of type int, int8, int16, int32, int64, uintptr, uint,
// uint8, uint32, or uint64, float32 or float64 by n. Returns an error if the
// item's value is not an integer, if it was not found, or if it is not
// possible to decrement it by n. To retrieve the decremented value, use one
// of the specialized methods, e.g. DecrementInt64.
func (c *cache) Decrement(k string, n int64) error {
	// TODO: Implement Increment and Decrement more cleanly.
	// (Cannot do Increment(k, n*-1) for uints.)
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return fmt.Errorf("Item not found")
	}
	switch v.Object.(type) {
	case int:
		v.Object = v.Object.(int) - int(n)
	case int8:
		v.Object = v.Object.(int8) - int8(n)
	case int16:
		v.Object = v.Object.(int16) - int16(n)
	case int32:
		v.Object = v.Object.(int32) - int32(n)
	case int64:
		v.Object = v.Object.(int64) - n
	case uint:
		v.Object = v.Object.(uint) - uint(n)
	case uintptr:
		v.Object = v.Object.(uintptr) - uintptr(n)
	case uint8:
		v.Object = v.Object.(uint8) - uint8(n)
	case uint16:
		v.Object = v.Object.(uint16) - uint16(n)
	case uint32:
		v.Object = v.Object.(uint32) - uint32(n)
	case uint64:
		v.Object = v.Object.(uint64) - uint64(n)
	case float32:
		v.Object = v.Object.(float32) - float32(n)
	case float64:
		v.Object = v.Object.(float64) - float64(n)
	default:
		c.storage.Unlock()
		return fmt.Errorf("The value for %s is not an integer", k)
	}
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nil
}

// Decrement an item of type float32 or float64 by n. Returns an error if the
// item's value is not floating point, if it was not found, or if it is not
// possible to decrement it by n. Pass a negative number to decrement the
// value. To retrieve the decremented value, use one of the specialized methods,
// e.g. DecrementFloat64.
func (c *cache) DecrementFloat(k string, n float64) error {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return fmt.Errorf("Item %s not found", k)
	}
	switch v.Object.(type) {
	case float32:
		v.Object = v.Object.(float32) - float32(n)
	case float64:
		v.Object = v.Object.(float64) - n
	default:
		c.storage.Unlock()
		return fmt.Errorf("The value for %s does not have type float32 or float64", k)
	}
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nil
}

// Decrement an item of type int by n. Returns an error if the item's value is
// not an int, or if it was not found. If there is no error, the decremented
// value is returned.
func (c *cache) DecrementInt(k string, n int) (int, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int", k)
	}
	nv := rv - n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Decrement an item of type int8 by n. Returns an error if the item's value is
// not an int8, or if it was not found. If there is no error, the decremented
// value is returned.
func (c *cache) DecrementInt8(k string, n int8) (int8, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int8)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int8", k)
	}
	nv := rv - n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Decrement an item of type int16 by n. Returns an error if the item's value is
// not an int16, or if it was not found. If there is no error, the decremented
// value is returned.
func (c *cache) DecrementInt16(k string, n int16) (int16, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int16)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int16", k)
	}
	nv := rv - n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Decrement an item of type int32 by n. Returns an error if the item's value is
// not an int32, or if it was not found. If there is no error, the decremented
// value is returned.
func (c *cache) DecrementInt32(k string, n int32) (int32, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int32)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int32", k)
	}
	nv := rv - n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Decrement an item of type int64 by n. Returns an error if the item's value is
// not an int64, or if it was not found. If there is no error, the decremented
// value is returned.
func (c *cache) DecrementInt64(k string, n int64) (int64, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(int64)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an int64", k)
	}
	nv := rv - n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Decrement an item of type uint by n. Returns an error if the item's value is
// not an uint, or if it was not found. If there is no error, the decremented
// value is returned.
func (c *cache) DecrementUint(k string, n uint) (uint, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint", k)
	}
	nv := rv - n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Decrement an item of type uintptr by n. Returns an error if the item's value
// is not an uintptr, or if it was not found. If there is no error, the
// decremented value is returned.
func (c *cache) DecrementUintptr(k string, n uintptr) (uintptr, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uintptr)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uintptr", k)
	}
	nv := rv - n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Decrement an item of type uint8 by n. Returns an error if the item's value is
// not an uint8, or if it was not found. If there is no error, the decremented
// value is returned.
func (c *cache) DecrementUint8(k string, n uint8) (uint8, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint8)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint8", k)
	}
	nv := rv - n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Decrement an item of type uint16 by n. Returns an error if the item's value
// is not an uint16, or if it was not found. If there is no error, the
// decremented value is returned.
func (c *cache) DecrementUint16(k string, n uint16) (uint16, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint16)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint16", k)
	}
	nv := rv - n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Decrement an item of type uint32 by n. Returns an error if the item's value
// is not an uint32, or if it was not found. If there is no error, the
// decremented value is returned.
func (c *cache) DecrementUint32(k string, n uint32) (uint32, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint32)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint32", k)
	}
	nv := rv - n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Decrement an item of type uint64 by n. Returns an error if the item's value
// is not an uint64, or if it was not found. If there is no error, the
// decremented value is returned.
func (c *cache) DecrementUint64(k string, n uint64) (uint64, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(uint64)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an uint64", k)
	}
	nv := rv - n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Decrement an item of type float32 by n. Returns an error if the item's value
// is not an float32, or if it was not found. If there is no error, the
// decremented value is returned.
func (c *cache) DecrementFloat32(k string, n float32) (float32, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(float32)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an float32", k)
	}
	nv := rv - n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Decrement an item of type float64 by n. Returns an error if the item's value
// is not an float64, or if it was not found. If there is no error, the
// decremented value is returned.
func (c *cache) DecrementFloat64(k string, n float64) (float64, error) {
	c.storage.Lock()
	v, found := c.storage.Get(k)
	if !found || v.Expired() {
		c.storage.Unlock()
		return 0, fmt.Errorf("Item %s not found", k)
	}
	rv, ok := v.Object.(float64)
	if !ok {
		c.storage.Unlock()
		return 0, fmt.Errorf("The value for %s is not an float64", k)
	}
	nv := rv - n
	v.Object = nv
	c.storage.Set(k, v)
	c.storage.Unlock()
	return nv, nil
}

// Delete an item from the cache. Does nothing if the key is not in the cache.
func (c *cache) Delete(k string) {
	c.storage.Lock()
	c.delete(k)
	c.storage.Unlock()
}

func (c *cache) delete(k string) {
	c.storage.Del(k)
}

type keyAndValue struct {
	key   string
	value interface{}
}


// Sets an (optional) function that is called with the key and value when an
// item has reached its refresh deadline from the cache.
func (c *cache) OnRefreshNeeded(f func(string)) {
	c.storage.Lock()
	c.onRefreshNeeded = f
	c.storage.Unlock()
}

// Delete all items from the cache.
func (c *cache) Flush() {
	c.storage.Flush()
}

func newCache(de time.Duration, s Storage, refreshWorkerCount int) *cache {
	if de == 0 {
		de = -1
	}

	c := &cache{
		defaultExpiration: de,
		storage:             s,
		refreshConcurrencyMap:make(map[string]bool),
		refreshKeys: make(chan string, 100),
	}
	for i := 1; i <= refreshWorkerCount; i++ {
		go c.refreshWorker(i, c.refreshKeys)
	}
	return c
}

// Return a new cache with a given default expiration duration and cleanup
// interval. If the expiration duration is less than one (or NoExpiration),
// the items in the cache never expire (by default), and must be deleted
// manually. If the cleanup interval is less than one, expired items are not
// deleted from the cache before calling c.DeleteExpired().
func New(defaultExpiration, cleanupInterval time.Duration, refreshWorkerCount int, storage Storage) *Cache {
	if storage.Type() == STORAGE_TYPE_MEMORY {
		c := newCache(defaultExpiration, storage, refreshWorkerCount)
		// This trick ensures that the janitor goroutine (which--granted it
		// was enabled--is running DeleteExpired on c forever) does not keep
		// the returned C object from being garbage collected. When it is
		// garbage collected, the finalizer stops the janitor goroutine, after
		// which c can be collected.
		C := &Cache{c}
		if cleanupInterval > 0 {
			runJanitor(storage.(*memoryStorage), cleanupInterval)
			runtime.SetFinalizer(storage, stopJanitor)
		}
		return C

	} else if storage.Type() == STORAGE_TYPE_REDIS {
		return &Cache{newCache(defaultExpiration, storage, refreshWorkerCount)}
	} else {
		panic("Unknown storage type")
	}
}
