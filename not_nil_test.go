// Copyright 2016 Qiang Xue. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MyInterface interface {
	Hello()
}

func TestNotNil(t *testing.T) {
	var v1 []int
	var v2 map[string]int
	var v3 *int
	var v4 interface{}
	var v5 MyInterface
	tests := []struct {
		tag   string
		value interface{}
		err   string
	}{
		{"t1", v1, "is required"},
		{"t2", v2, "is required"},
		{"t3", v3, "is required"},
		{"t4", v4, "is required"},
		{"t5", v5, "is required"},
		{"t6", "", ""},
		{"t7", 0, ""},
	}

	for _, test := range tests {
		r := NotNil
		err := r.Validate(nil, test.value)
		assertError(t, test.err, err, test.tag)
	}
}

func Test_notNilRule_Error(t *testing.T) {
	r := NotNil
	assert.Equal(t, "is required", r.Validate(nil, nil).Error())
	r2 := r.Error("123")
	assert.Equal(t, "is required", r.Validate(nil, nil).Error())
	assert.Equal(t, "123", r2.err.Message())
}

func TestNotNilRule_ErrorObject(t *testing.T) {
	r := NotNil

	err := NewError("code", "abc")
	r = r.ErrorObject(err)

	assert.Equal(t, err, r.err)
	assert.Equal(t, err.Code(), r.err.Code())
	assert.Equal(t, err.Message(), r.err.Message())
	assert.NotEqual(t, err, NotNil.err)
}

func TestNotNilWithFunctionAndChannel(t *testing.T) {
	// Test function type
	var fn func()
	err := NotNil.Validate(nil, fn)
	assert.NotNil(t, err, "nil function should fail")

	fn = func() {}
	err = NotNil.Validate(nil, fn)
	assert.Nil(t, err, "non-nil function should pass")

	// Test channel type
	var ch chan int
	err = NotNil.Validate(nil, ch)
	assert.NotNil(t, err, "nil channel should fail")

	ch = make(chan int)
	err = NotNil.Validate(nil, ch)
	assert.Nil(t, err, "non-nil channel should pass")
	close(ch)

	// Test buffered channel
	bufCh := make(chan string, 5)
	err = NotNil.Validate(nil, bufCh)
	assert.Nil(t, err, "buffered channel should pass")
	close(bufCh)
}

func TestNotNilWithComplexTypes(t *testing.T) {
	// Test struct (non-pointer) - should pass since it's not nil
	type Person struct {
		Name string
	}
	p := Person{Name: "John"}
	err := NotNil.Validate(nil, p)
	assert.Nil(t, err, "non-pointer struct should pass")

	// Test array (non-pointer) - should pass
	arr := [3]int{1, 2, 3}
	err = NotNil.Validate(nil, arr)
	assert.Nil(t, err, "non-pointer array should pass")

	// Test difference between nil slice and empty slice
	var nilSlice []int
	err = NotNil.Validate(nil, nilSlice)
	assert.NotNil(t, err, "nil slice should fail")

	emptySlice := []int{}
	err = NotNil.Validate(nil, emptySlice)
	assert.Nil(t, err, "empty but non-nil slice should pass")

	// Test difference between nil map and empty map
	var nilMap map[string]int
	err = NotNil.Validate(nil, nilMap)
	assert.NotNil(t, err, "nil map should fail")

	emptyMap := map[string]int{}
	err = NotNil.Validate(nil, emptyMap)
	assert.Nil(t, err, "empty but non-nil map should pass")
}

func TestNotNilWithPointerToPointer(t *testing.T) {
	// Test multi-level pointers
	var p1 *int
	err := NotNil.Validate(nil, p1)
	assert.NotNil(t, err, "nil pointer should fail")

	var p2 **int
	err = NotNil.Validate(nil, p2)
	assert.NotNil(t, err, "nil pointer to pointer should fail")

	i := 42
	p1 = &i
	p2 = &p1
	err = NotNil.Validate(nil, p2)
	assert.Nil(t, err, "non-nil pointer to pointer should pass")

	// Test: pointer to nil pointer (outer is not nil, inner is nil)
	var innerPtr *int
	outerPtr := &innerPtr
	err = NotNil.Validate(nil, outerPtr)
	assert.NotNil(t, err, "pointer to nil pointer should fail (indirection goes to nil)")
}

func TestNotNilWithInterface(t *testing.T) {
	// Test nil interface
	var i interface{}
	err := NotNil.Validate(nil, i)
	assert.NotNil(t, err, "nil interface should fail")

	// Test interface containing nil pointer
	var ptr *int
	i = ptr
	err = NotNil.Validate(nil, i)
	assert.NotNil(t, err, "interface containing nil pointer should fail")

	// Test interface containing non-nil value
	i = 42
	err = NotNil.Validate(nil, i)
	assert.Nil(t, err, "interface containing value should pass")

	// Test interface containing non-nil pointer
	value := 42
	i = &value
	err = NotNil.Validate(nil, i)
	assert.Nil(t, err, "interface containing non-nil pointer should pass")
}

func TestNotNilWithContext(t *testing.T) {
	// Test using context and options
	ctx := context.Background()

	var nilSlice []int
	err := NotNil.Validate(ctx, nilSlice)
	assert.NotNil(t, err)

	slice := []int{1, 2, 3}
	err = NotNil.Validate(ctx, slice)
	assert.Nil(t, err)
}

func TestNotNilErrorMessages(t *testing.T) {
	// Test default error message
	var v1 []int
	err := NotNil.Validate(nil, v1)
	if assert.NotNil(t, err) {
		assert.Equal(t, "is required", err.Error())
	}

	// Test custom error message
	customRule := NotNil.Error("value must not be nil")
	err = customRule.Validate(nil, v1)
	if assert.NotNil(t, err) {
		assert.Equal(t, "value must not be nil", err.Error())
	}

	// Ensure original rule is not modified
	err = NotNil.Validate(nil, v1)
	if assert.NotNil(t, err) {
		assert.Equal(t, "is required", err.Error())
	}
}
