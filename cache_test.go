package cache

import (
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

type TestStruct struct {
	Num      int
	Children []*TestStruct
}

func TestCache(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())

	a, found := tc.Get("a")
	if found || a != nil {
		t.Error("Getting A found value that shouldn't exist:", a)
	}

	b, found := tc.Get("b")
	if found || b != nil {
		t.Error("Getting B found value that shouldn't exist:", b)
	}

	c, found := tc.Get("c")
	if found || c != nil {
		t.Error("Getting C found value that shouldn't exist:", c)
	}

	tc.Set("a", 1, DefaultExpiration, NoRefreshDeadline)
	tc.Set("b", "b", DefaultExpiration, NoRefreshDeadline)
	tc.Set("c", 3.5, DefaultExpiration, NoRefreshDeadline)

	x, found := tc.Get("a")
	if !found {
		t.Error("a was not found while getting a2")
	}
	if x == nil {
		t.Error("x for a is nil")
	} else if a2 := x.(int); a2 + 2 != 3 {
		t.Error("a2 (which should be 1) plus 2 does not equal 3; value:", a2)
	}

	x, found = tc.Get("b")
	if !found {
		t.Error("b was not found while getting b2")
	}
	if x == nil {
		t.Error("x for b is nil")
	} else if b2 := x.(string); b2 + "B" != "bB" {
		t.Error("b2 (which should be b) plus B does not equal bB; value:", b2)
	}

	x, found = tc.Get("c")
	if !found {
		t.Error("c was not found while getting c2")
	}
	if x == nil {
		t.Error("x for c is nil")
	} else if c2 := x.(float64); c2 + 1.2 != 4.7 {
		t.Error("c2 (which should be 3.5) plus 1.2 does not equal 4.7; value:", c2)
	}
}

func TestCacheTimes(t *testing.T) {
	var found bool

	tc := New(50 * time.Millisecond, 1 * time.Millisecond, 0, MemoryStorage())
	tc.Set("a", 1, DefaultExpiration, NoRefreshDeadline)
	tc.Set("b", 2, NoExpiration, NoRefreshDeadline)
	tc.Set("c", 3, 20 * time.Millisecond, NoRefreshDeadline)
	tc.Set("d", 4, 70 * time.Millisecond, NoRefreshDeadline)

	<-time.After(25 * time.Millisecond)
	_, found = tc.Get("c")
	if found {
		t.Error("Found c when it should have been automatically deleted")
	}

	<-time.After(30 * time.Millisecond)
	_, found = tc.Get("a")
	if found {
		t.Error("Found a when it should have been automatically deleted")
	}

	_, found = tc.Get("b")
	if !found {
		t.Error("Did not find b even though it was set to never expire")
	}

	_, found = tc.Get("d")
	if !found {
		t.Error("Did not find d even though it was set to expire later than the default")
	}

	<-time.After(20 * time.Millisecond)
	_, found = tc.Get("d")
	if found {
		t.Error("Found d when it should have been automatically deleted (later than the default)")
	}
}

func TestStorePointerToStruct(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("foo", &TestStruct{Num: 1}, DefaultExpiration, NoRefreshDeadline)
	x, found := tc.Get("foo")
	if !found {
		t.Fatal("*TestStruct was not found for foo")
	}
	foo := x.(*TestStruct)
	foo.Num++

	y, found := tc.Get("foo")
	if !found {
		t.Fatal("*TestStruct was not found for foo (second time)")
	}
	bar := y.(*TestStruct)
	if bar.Num != 2 {
		t.Fatal("TestStruct.Num is not 2")
	}
}

func TestIncrementWithInt(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tint", 1, DefaultExpiration, NoRefreshDeadline)
	err := tc.Increment("tint", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	x, found := tc.Get("tint")
	if !found {
		t.Error("tint was not found")
	}
	if x.(int) != 3 {
		t.Error("tint is not 3:", x)
	}
}

func TestIncrementWithInt8(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tint8", int8(1), DefaultExpiration, NoRefreshDeadline)
	err := tc.Increment("tint8", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	x, found := tc.Get("tint8")
	if !found {
		t.Error("tint8 was not found")
	}
	if x.(int8) != 3 {
		t.Error("tint8 is not 3:", x)
	}
}

func TestIncrementWithInt16(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tint16", int16(1), DefaultExpiration, NoRefreshDeadline)
	err := tc.Increment("tint16", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	x, found := tc.Get("tint16")
	if !found {
		t.Error("tint16 was not found")
	}
	if x.(int16) != 3 {
		t.Error("tint16 is not 3:", x)
	}
}

func TestIncrementWithInt32(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tint32", int32(1), DefaultExpiration, NoRefreshDeadline)
	err := tc.Increment("tint32", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	x, found := tc.Get("tint32")
	if !found {
		t.Error("tint32 was not found")
	}
	if x.(int32) != 3 {
		t.Error("tint32 is not 3:", x)
	}
}

func TestIncrementWithInt64(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tint64", int64(1), DefaultExpiration, NoRefreshDeadline)
	err := tc.Increment("tint64", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	x, found := tc.Get("tint64")
	if !found {
		t.Error("tint64 was not found")
	}
	if x.(int64) != 3 {
		t.Error("tint64 is not 3:", x)
	}
}

func TestIncrementWithUint(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tuint", uint(1), DefaultExpiration, NoRefreshDeadline)
	err := tc.Increment("tuint", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	x, found := tc.Get("tuint")
	if !found {
		t.Error("tuint was not found")
	}
	if x.(uint) != 3 {
		t.Error("tuint is not 3:", x)
	}
}

func TestIncrementWithUintptr(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tuintptr", uintptr(1), DefaultExpiration, NoRefreshDeadline)
	err := tc.Increment("tuintptr", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}

	x, found := tc.Get("tuintptr")
	if !found {
		t.Error("tuintptr was not found")
	}
	if x.(uintptr) != 3 {
		t.Error("tuintptr is not 3:", x)
	}
}

func TestIncrementWithUint8(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tuint8", uint8(1), DefaultExpiration, NoRefreshDeadline)
	err := tc.Increment("tuint8", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	x, found := tc.Get("tuint8")
	if !found {
		t.Error("tuint8 was not found")
	}
	if x.(uint8) != 3 {
		t.Error("tuint8 is not 3:", x)
	}
}

func TestIncrementWithUint16(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tuint16", uint16(1), DefaultExpiration, NoRefreshDeadline)
	err := tc.Increment("tuint16", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}

	x, found := tc.Get("tuint16")
	if !found {
		t.Error("tuint16 was not found")
	}
	if x.(uint16) != 3 {
		t.Error("tuint16 is not 3:", x)
	}
}

func TestIncrementWithUint32(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tuint32", uint32(1), DefaultExpiration, NoRefreshDeadline)
	err := tc.Increment("tuint32", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	x, found := tc.Get("tuint32")
	if !found {
		t.Error("tuint32 was not found")
	}
	if x.(uint32) != 3 {
		t.Error("tuint32 is not 3:", x)
	}
}

func TestIncrementWithUint64(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tuint64", uint64(1), DefaultExpiration, NoRefreshDeadline)
	err := tc.Increment("tuint64", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}

	x, found := tc.Get("tuint64")
	if !found {
		t.Error("tuint64 was not found")
	}
	if x.(uint64) != 3 {
		t.Error("tuint64 is not 3:", x)
	}
}

func TestIncrementWithFloat32(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("float32", float32(1.5), DefaultExpiration, NoRefreshDeadline)
	err := tc.Increment("float32", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	x, found := tc.Get("float32")
	if !found {
		t.Error("float32 was not found")
	}
	if x.(float32) != 3.5 {
		t.Error("float32 is not 3.5:", x)
	}
}

func TestIncrementWithFloat64(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("float64", float64(1.5), DefaultExpiration, NoRefreshDeadline)
	err := tc.Increment("float64", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	x, found := tc.Get("float64")
	if !found {
		t.Error("float64 was not found")
	}
	if x.(float64) != 3.5 {
		t.Error("float64 is not 3.5:", x)
	}
}

func TestIncrementFloatWithFloat32(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("float32", float32(1.5), DefaultExpiration, NoRefreshDeadline)
	err := tc.IncrementFloat("float32", 2)
	if err != nil {
		t.Error("Error incrementfloating:", err)
	}
	x, found := tc.Get("float32")
	if !found {
		t.Error("float32 was not found")
	}
	if x.(float32) != 3.5 {
		t.Error("float32 is not 3.5:", x)
	}
}

func TestIncrementFloatWithFloat64(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("float64", float64(1.5), DefaultExpiration, NoRefreshDeadline)
	err := tc.IncrementFloat("float64", 2)
	if err != nil {
		t.Error("Error incrementfloating:", err)
	}
	x, found := tc.Get("float64")
	if !found {
		t.Error("float64 was not found")
	}
	if x.(float64) != 3.5 {
		t.Error("float64 is not 3.5:", x)
	}
}

func TestDecrementWithInt(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("int", int(5), DefaultExpiration, NoRefreshDeadline)
	err := tc.Decrement("int", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	x, found := tc.Get("int")
	if !found {
		t.Error("int was not found")
	}
	if x.(int) != 3 {
		t.Error("int is not 3:", x)
	}
}

func TestDecrementWithInt8(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("int8", int8(5), DefaultExpiration, NoRefreshDeadline)
	err := tc.Decrement("int8", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	x, found := tc.Get("int8")
	if !found {
		t.Error("int8 was not found")
	}
	if x.(int8) != 3 {
		t.Error("int8 is not 3:", x)
	}
}

func TestDecrementWithInt16(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("int16", int16(5), DefaultExpiration, NoRefreshDeadline)
	err := tc.Decrement("int16", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	x, found := tc.Get("int16")
	if !found {
		t.Error("int16 was not found")
	}
	if x.(int16) != 3 {
		t.Error("int16 is not 3:", x)
	}
}

func TestDecrementWithInt32(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("int32", int32(5), DefaultExpiration, NoRefreshDeadline)
	err := tc.Decrement("int32", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	x, found := tc.Get("int32")
	if !found {
		t.Error("int32 was not found")
	}
	if x.(int32) != 3 {
		t.Error("int32 is not 3:", x)
	}
}

func TestDecrementWithInt64(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("int64", int64(5), DefaultExpiration, NoRefreshDeadline)
	err := tc.Decrement("int64", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	x, found := tc.Get("int64")
	if !found {
		t.Error("int64 was not found")
	}
	if x.(int64) != 3 {
		t.Error("int64 is not 3:", x)
	}
}

func TestDecrementWithUint(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("uint", uint(5), DefaultExpiration, NoRefreshDeadline)
	err := tc.Decrement("uint", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	x, found := tc.Get("uint")
	if !found {
		t.Error("uint was not found")
	}
	if x.(uint) != 3 {
		t.Error("uint is not 3:", x)
	}
}

func TestDecrementWithUintptr(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("uintptr", uintptr(5), DefaultExpiration, NoRefreshDeadline)
	err := tc.Decrement("uintptr", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	x, found := tc.Get("uintptr")
	if !found {
		t.Error("uintptr was not found")
	}
	if x.(uintptr) != 3 {
		t.Error("uintptr is not 3:", x)
	}
}

func TestDecrementWithUint8(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("uint8", uint8(5), DefaultExpiration, NoRefreshDeadline)
	err := tc.Decrement("uint8", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	x, found := tc.Get("uint8")
	if !found {
		t.Error("uint8 was not found")
	}
	if x.(uint8) != 3 {
		t.Error("uint8 is not 3:", x)
	}
}

func TestDecrementWithUint16(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("uint16", uint16(5), DefaultExpiration, NoRefreshDeadline)
	err := tc.Decrement("uint16", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	x, found := tc.Get("uint16")
	if !found {
		t.Error("uint16 was not found")
	}
	if x.(uint16) != 3 {
		t.Error("uint16 is not 3:", x)
	}
}

func TestDecrementWithUint32(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("uint32", uint32(5), DefaultExpiration, NoRefreshDeadline)
	err := tc.Decrement("uint32", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	x, found := tc.Get("uint32")
	if !found {
		t.Error("uint32 was not found")
	}
	if x.(uint32) != 3 {
		t.Error("uint32 is not 3:", x)
	}
}

func TestDecrementWithUint64(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("uint64", uint64(5), DefaultExpiration, NoRefreshDeadline)
	err := tc.Decrement("uint64", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	x, found := tc.Get("uint64")
	if !found {
		t.Error("uint64 was not found")
	}
	if x.(uint64) != 3 {
		t.Error("uint64 is not 3:", x)
	}
}

func TestDecrementWithFloat32(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("float32", float32(5.5), DefaultExpiration, NoRefreshDeadline)
	err := tc.Decrement("float32", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	x, found := tc.Get("float32")
	if !found {
		t.Error("float32 was not found")
	}
	if x.(float32) != 3.5 {
		t.Error("float32 is not 3:", x)
	}
}

func TestDecrementWithFloat64(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("float64", float64(5.5), DefaultExpiration, NoRefreshDeadline)
	err := tc.Decrement("float64", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	x, found := tc.Get("float64")
	if !found {
		t.Error("float64 was not found")
	}
	if x.(float64) != 3.5 {
		t.Error("float64 is not 3:", x)
	}
}

func TestDecrementFloatWithFloat32(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("float32", float32(5.5), DefaultExpiration, NoRefreshDeadline)
	err := tc.DecrementFloat("float32", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	x, found := tc.Get("float32")
	if !found {
		t.Error("float32 was not found")
	}
	if x.(float32) != 3.5 {
		t.Error("float32 is not 3:", x)
	}
}

func TestDecrementFloatWithFloat64(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("float64", float64(5.5), DefaultExpiration, NoRefreshDeadline)
	err := tc.DecrementFloat("float64", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	x, found := tc.Get("float64")
	if !found {
		t.Error("float64 was not found")
	}
	if x.(float64) != 3.5 {
		t.Error("float64 is not 3:", x)
	}
}

func TestIncrementInt(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tint", 1, DefaultExpiration, NoRefreshDeadline)
	n, err := tc.IncrementInt("tint", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("tint")
	if !found {
		t.Error("tint was not found")
	}
	if x.(int) != 3 {
		t.Error("tint is not 3:", x)
	}
}

func TestIncrementInt8(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tint8", int8(1), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.IncrementInt8("tint8", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("tint8")
	if !found {
		t.Error("tint8 was not found")
	}
	if x.(int8) != 3 {
		t.Error("tint8 is not 3:", x)
	}
}

func TestIncrementInt16(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tint16", int16(1), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.IncrementInt16("tint16", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("tint16")
	if !found {
		t.Error("tint16 was not found")
	}
	if x.(int16) != 3 {
		t.Error("tint16 is not 3:", x)
	}
}

func TestIncrementInt32(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tint32", int32(1), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.IncrementInt32("tint32", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("tint32")
	if !found {
		t.Error("tint32 was not found")
	}
	if x.(int32) != 3 {
		t.Error("tint32 is not 3:", x)
	}
}

func TestIncrementInt64(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tint64", int64(1), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.IncrementInt64("tint64", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("tint64")
	if !found {
		t.Error("tint64 was not found")
	}
	if x.(int64) != 3 {
		t.Error("tint64 is not 3:", x)
	}
}

func TestIncrementUint(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tuint", uint(1), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.IncrementUint("tuint", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("tuint")
	if !found {
		t.Error("tuint was not found")
	}
	if x.(uint) != 3 {
		t.Error("tuint is not 3:", x)
	}
}

func TestIncrementUintptr(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tuintptr", uintptr(1), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.IncrementUintptr("tuintptr", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("tuintptr")
	if !found {
		t.Error("tuintptr was not found")
	}
	if x.(uintptr) != 3 {
		t.Error("tuintptr is not 3:", x)
	}
}

func TestIncrementUint8(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tuint8", uint8(1), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.IncrementUint8("tuint8", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("tuint8")
	if !found {
		t.Error("tuint8 was not found")
	}
	if x.(uint8) != 3 {
		t.Error("tuint8 is not 3:", x)
	}
}

func TestIncrementUint16(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tuint16", uint16(1), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.IncrementUint16("tuint16", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("tuint16")
	if !found {
		t.Error("tuint16 was not found")
	}
	if x.(uint16) != 3 {
		t.Error("tuint16 is not 3:", x)
	}
}

func TestIncrementUint32(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tuint32", uint32(1), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.IncrementUint32("tuint32", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("tuint32")
	if !found {
		t.Error("tuint32 was not found")
	}
	if x.(uint32) != 3 {
		t.Error("tuint32 is not 3:", x)
	}
}

func TestIncrementUint64(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("tuint64", uint64(1), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.IncrementUint64("tuint64", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("tuint64")
	if !found {
		t.Error("tuint64 was not found")
	}
	if x.(uint64) != 3 {
		t.Error("tuint64 is not 3:", x)
	}
}

func TestIncrementFloat32(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("float32", float32(1.5), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.IncrementFloat32("float32", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	if n != 3.5 {
		t.Error("Returned number is not 3.5:", n)
	}
	x, found := tc.Get("float32")
	if !found {
		t.Error("float32 was not found")
	}
	if x.(float32) != 3.5 {
		t.Error("float32 is not 3.5:", x)
	}
}

func TestIncrementFloat64(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("float64", float64(1.5), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.IncrementFloat64("float64", 2)
	if err != nil {
		t.Error("Error incrementing:", err)
	}
	if n != 3.5 {
		t.Error("Returned number is not 3.5:", n)
	}
	x, found := tc.Get("float64")
	if !found {
		t.Error("float64 was not found")
	}
	if x.(float64) != 3.5 {
		t.Error("float64 is not 3.5:", x)
	}
}

func TestDecrementInt8(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("int8", int8(5), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.DecrementInt8("int8", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("int8")
	if !found {
		t.Error("int8 was not found")
	}
	if x.(int8) != 3 {
		t.Error("int8 is not 3:", x)
	}
}

func TestDecrementInt16(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("int16", int16(5), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.DecrementInt16("int16", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("int16")
	if !found {
		t.Error("int16 was not found")
	}
	if x.(int16) != 3 {
		t.Error("int16 is not 3:", x)
	}
}

func TestDecrementInt32(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("int32", int32(5), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.DecrementInt32("int32", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("int32")
	if !found {
		t.Error("int32 was not found")
	}
	if x.(int32) != 3 {
		t.Error("int32 is not 3:", x)
	}
}

func TestDecrementInt64(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("int64", int64(5), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.DecrementInt64("int64", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("int64")
	if !found {
		t.Error("int64 was not found")
	}
	if x.(int64) != 3 {
		t.Error("int64 is not 3:", x)
	}
}

func TestDecrementUint(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("uint", uint(5), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.DecrementUint("uint", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("uint")
	if !found {
		t.Error("uint was not found")
	}
	if x.(uint) != 3 {
		t.Error("uint is not 3:", x)
	}
}

func TestDecrementUintptr(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("uintptr", uintptr(5), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.DecrementUintptr("uintptr", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("uintptr")
	if !found {
		t.Error("uintptr was not found")
	}
	if x.(uintptr) != 3 {
		t.Error("uintptr is not 3:", x)
	}
}

func TestDecrementUint8(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("uint8", uint8(5), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.DecrementUint8("uint8", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("uint8")
	if !found {
		t.Error("uint8 was not found")
	}
	if x.(uint8) != 3 {
		t.Error("uint8 is not 3:", x)
	}
}

func TestDecrementUint16(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("uint16", uint16(5), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.DecrementUint16("uint16", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("uint16")
	if !found {
		t.Error("uint16 was not found")
	}
	if x.(uint16) != 3 {
		t.Error("uint16 is not 3:", x)
	}
}

func TestDecrementUint32(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("uint32", uint32(5), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.DecrementUint32("uint32", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("uint32")
	if !found {
		t.Error("uint32 was not found")
	}
	if x.(uint32) != 3 {
		t.Error("uint32 is not 3:", x)
	}
}

func TestDecrementUint64(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("uint64", uint64(5), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.DecrementUint64("uint64", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("uint64")
	if !found {
		t.Error("uint64 was not found")
	}
	if x.(uint64) != 3 {
		t.Error("uint64 is not 3:", x)
	}
}

func TestDecrementFloat32(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("float32", float32(5), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.DecrementFloat32("float32", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("float32")
	if !found {
		t.Error("float32 was not found")
	}
	if x.(float32) != 3 {
		t.Error("float32 is not 3:", x)
	}
}

func TestDecrementFloat64(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("float64", float64(5), DefaultExpiration, NoRefreshDeadline)
	n, err := tc.DecrementFloat64("float64", 2)
	if err != nil {
		t.Error("Error decrementing:", err)
	}
	if n != 3 {
		t.Error("Returned number is not 3:", n)
	}
	x, found := tc.Get("float64")
	if !found {
		t.Error("float64 was not found")
	}
	if x.(float64) != 3 {
		t.Error("float64 is not 3:", x)
	}
}

func TestAdd(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	err := tc.Add("foo", "bar", DefaultExpiration, NoRefreshDeadline)
	if err != nil {
		t.Error("Couldn't add foo even though it shouldn't exist")
	}
	err = tc.Add("foo", "baz", DefaultExpiration, NoRefreshDeadline)
	if err == nil {
		t.Error("Successfully added another foo when it should have returned an error")
	}
}

func TestReplace(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	err := tc.Replace("foo", "bar", DefaultExpiration, NoRefreshDeadline)
	if err == nil {
		t.Error("Replaced foo when it shouldn't exist")
	}
	tc.Set("foo", "bar", DefaultExpiration, NoRefreshDeadline)
	err = tc.Replace("foo", "bar", DefaultExpiration, NoRefreshDeadline)
	if err != nil {
		t.Error("Couldn't replace existing key foo")
	}
}

func TestDelete(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("foo", "bar", DefaultExpiration, NoRefreshDeadline)
	tc.Delete("foo")
	x, found := tc.Get("foo")
	if found {
		t.Error("foo was found, but it should have been deleted")
	}
	if x != nil {
		t.Error("x is not nil:", x)
	}
}

func TestFlush(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("foo", "bar", DefaultExpiration, NoRefreshDeadline)
	tc.Set("baz", "yes", DefaultExpiration, NoRefreshDeadline)
	tc.Flush()
	x, found := tc.Get("foo")
	if found {
		t.Error("foo was found, but it should have been deleted")
	}
	if x != nil {
		t.Error("x is not nil:", x)
	}
	x, found = tc.Get("baz")
	if found {
		t.Error("baz was found, but it should have been deleted")
	}
	if x != nil {
		t.Error("x is not nil:", x)
	}
}

func TestIncrementOverflowInt(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("int8", int8(127), DefaultExpiration, NoRefreshDeadline)
	err := tc.Increment("int8", 1)
	if err != nil {
		t.Error("Error incrementing int8:", err)
	}
	x, _ := tc.Get("int8")
	int8 := x.(int8)
	if int8 != -128 {
		t.Error("int8 did not overflow as expected; value:", int8)
	}

}

func TestIncrementOverflowUint(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("uint8", uint8(255), DefaultExpiration, NoRefreshDeadline)
	err := tc.Increment("uint8", 1)
	if err != nil {
		t.Error("Error incrementing int8:", err)
	}
	x, _ := tc.Get("uint8")
	uint8 := x.(uint8)
	if uint8 != 0 {
		t.Error("uint8 did not overflow as expected; value:", uint8)
	}
}

func TestDecrementUnderflowUint(t *testing.T) {
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("uint8", uint8(0), DefaultExpiration, NoRefreshDeadline)
	err := tc.Decrement("uint8", 1)
	if err != nil {
		t.Error("Error decrementing int8:", err)
	}
	x, _ := tc.Get("uint8")
	uint8 := x.(uint8)
	if uint8 != 255 {
		t.Error("uint8 did not underflow as expected; value:", uint8)
	}
}

func BenchmarkCacheGetExpiring(b *testing.B) {
	benchmarkCacheGet(b, 5 * time.Minute)
}

func BenchmarkCacheGetNotExpiring(b *testing.B) {
	benchmarkCacheGet(b, NoExpiration)
}

func benchmarkCacheGet(b *testing.B, exp time.Duration) {
	b.StopTimer()
	tc := New(exp, 0, 0, MemoryStorage())
	tc.Set("foo", "bar", DefaultExpiration, NoRefreshDeadline)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Get("foo")
	}
}

func BenchmarkRWMutexMapGet(b *testing.B) {
	b.StopTimer()
	m := map[string]string{
		"foo": "bar",
	}
	mu := sync.RWMutex{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		_, _ = m["foo"]
		mu.RUnlock()
	}
}

func BenchmarkRWMutexInterfaceMapGetStruct(b *testing.B) {
	b.StopTimer()
	s := struct{ name string }{name: "foo"}
	m := map[interface{}]string{
		s: "bar",
	}
	mu := sync.RWMutex{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		_, _ = m[s]
		mu.RUnlock()
	}
}

func BenchmarkRWMutexInterfaceMapGetString(b *testing.B) {
	b.StopTimer()
	m := map[interface{}]string{
		"foo": "bar",
	}
	mu := sync.RWMutex{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		_, _ = m["foo"]
		mu.RUnlock()
	}
}

func BenchmarkCacheGetConcurrentExpiring(b *testing.B) {
	benchmarkCacheGetConcurrent(b, 5 * time.Minute)
}

func BenchmarkCacheGetConcurrentNotExpiring(b *testing.B) {
	benchmarkCacheGetConcurrent(b, NoExpiration)
}

func benchmarkCacheGetConcurrent(b *testing.B, exp time.Duration) {
	b.StopTimer()
	tc := New(exp, 0, 0, MemoryStorage())
	tc.Set("foo", "bar", DefaultExpiration, NoRefreshDeadline)
	wg := new(sync.WaitGroup)
	workers := runtime.NumCPU()
	each := b.N / workers
	wg.Add(workers)
	b.StartTimer()
	for i := 0; i < workers; i++ {
		go func() {
			for j := 0; j < each; j++ {
				tc.Get("foo")
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkRWMutexMapGetConcurrent(b *testing.B) {
	b.StopTimer()
	m := map[string]string{
		"foo": "bar",
	}
	mu := sync.RWMutex{}
	wg := new(sync.WaitGroup)
	workers := runtime.NumCPU()
	each := b.N / workers
	wg.Add(workers)
	b.StartTimer()
	for i := 0; i < workers; i++ {
		go func() {
			for j := 0; j < each; j++ {
				mu.RLock()
				_, _ = m["foo"]
				mu.RUnlock()
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkCacheGetManyConcurrentExpiring(b *testing.B) {
	benchmarkCacheGetManyConcurrent(b, 5 * time.Minute)
}

func BenchmarkCacheGetManyConcurrentNotExpiring(b *testing.B) {
	benchmarkCacheGetManyConcurrent(b, NoExpiration)
}

func benchmarkCacheGetManyConcurrent(b *testing.B, exp time.Duration) {
	// This is the same as BenchmarkCacheGetConcurrent, but its result
	// can be compared against BenchmarkShardedCacheGetManyConcurrent
	// in sharded_test.go.
	b.StopTimer()
	n := 10000
	tc := New(exp, 0, 0, MemoryStorage())
	keys := make([]string, n)
	for i := 0; i < n; i++ {
		k := "foo" + strconv.Itoa(n)
		keys[i] = k
		tc.Set(k, "bar", DefaultExpiration, NoRefreshDeadline)
	}
	each := b.N / n
	wg := new(sync.WaitGroup)
	wg.Add(n)
	for _, v := range keys {
		go func() {
			for j := 0; j < each; j++ {
				tc.Get(v)
			}
			wg.Done()
		}()
	}
	b.StartTimer()
	wg.Wait()
}

func BenchmarkCacheSetExpiring(b *testing.B) {
	benchmarkCacheSet(b, 5 * time.Minute)
}

func BenchmarkCacheSetNotExpiring(b *testing.B) {
	benchmarkCacheSet(b, NoExpiration)
}

func benchmarkCacheSet(b *testing.B, exp time.Duration) {
	b.StopTimer()
	tc := New(exp, 0, 0, MemoryStorage())
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Set("foo", "bar", DefaultExpiration, NoRefreshDeadline)
	}
}

func BenchmarkRWMutexMapSet(b *testing.B) {
	b.StopTimer()
	m := map[string]string{}
	mu := sync.RWMutex{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m["foo"] = "bar"
		mu.Unlock()
	}
}

func BenchmarkCacheSetDelete(b *testing.B) {
	b.StopTimer()
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Set("foo", "bar", DefaultExpiration, NoRefreshDeadline)
		tc.Delete("foo")
	}
}

func BenchmarkRWMutexMapSetDelete(b *testing.B) {
	b.StopTimer()
	m := map[string]string{}
	mu := sync.RWMutex{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m["foo"] = "bar"
		mu.Unlock()
		mu.Lock()
		delete(m, "foo")
		mu.Unlock()
	}
}

func BenchmarkCacheSetDeleteSingleLock(b *testing.B) {
	b.StopTimer()
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.storage.Lock()
		tc.set("foo", "bar", DefaultExpiration, NoRefreshDeadline)
		tc.delete("foo")
		tc.storage.Unlock()
	}
}

func BenchmarkRWMutexMapSetDeleteSingleLock(b *testing.B) {
	b.StopTimer()
	m := map[string]string{}
	mu := sync.RWMutex{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m["foo"] = "bar"
		delete(m, "foo")
		mu.Unlock()
	}
}

func BenchmarkIncrementInt(b *testing.B) {
	b.StopTimer()
	tc := New(DefaultExpiration, 0, 0, MemoryStorage())
	tc.Set("foo", 0, DefaultExpiration, NoRefreshDeadline)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.IncrementInt("foo", 1)
	}
}
