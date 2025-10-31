// Copyright 2016 Qiang Xue. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package validation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	slice := []String123{String123("abc"), String123("123"), String123("xyz")}
	mp := map[string]String123{"c": String123("abc"), "b": String123("123"), "a": String123("xyz")}
	var ptr *string
	tests := []struct {
		tag   string
		value interface{}
		err   string
	}{
		{"t1", 123, ""},
		{"t2", String123("123"), ""},
		{"t3", String123("abc"), "error 123"},
		{"t4", []String123{}, ""},
		{"t4.1", []StringValidateContext{}, ""},
		{"t4.2", map[string]StringValidateContext{}, ""},
		{"t5", slice, "0: error 123; 2: error 123."},
		{"t6", &slice, "0: error 123; 2: error 123."},
		{"t8", mp, "a: error 123; c: error 123."},
		{"t9", &mp, "a: error 123; c: error 123."},
		{"t10", map[string]String123{}, ""},
		{"t11", ptr, ""},
	}
	for _, test := range tests {
		err := ValidateWithContext(nil, test.value)
		assertError(t, test.err, err, test.tag)
	}

	// with rules
	err := ValidateWithContext(nil, "123", &validateAbc{}, &validateXyz{})
	assert.EqualError(t, err, "error abc")
	err = ValidateWithContext(nil, "abc", &validateAbc{}, &validateXyz{})
	assert.EqualError(t, err, "error xyz")
	err = ValidateWithContext(nil, "abcxyz", &validateAbc{}, &validateXyz{})
	assert.NoError(t, err)

	err = ValidateWithContext(nil, "123", &validateAbc{}, Skip, &validateXyz{})
	assert.EqualError(t, err, "error abc")
	err = ValidateWithContext(nil, "abc", &validateAbc{}, Skip, &validateXyz{})
	assert.NoError(t, err)

	err = ValidateWithContext(nil, "123", &validateAbc{}, Skip.When(true), &validateXyz{})
	assert.EqualError(t, err, "error abc")
	err = ValidateWithContext(nil, "abc", &validateAbc{}, Skip.When(true), &validateXyz{})
	assert.NoError(t, err)

	err = ValidateWithContext(nil, "123", &validateAbc{}, Skip.When(false), &validateXyz{})
	assert.EqualError(t, err, "error abc")
	err = ValidateWithContext(nil, "abc", &validateAbc{}, Skip.When(false), &validateXyz{})
	assert.EqualError(t, err, "error xyz")
}

func stringEqual(str string) RuleFunc {
	return func(_context context.Context, value interface{}) error {
		s, _ := value.(string)
		if s != str {
			return errors.New("unexpected string")
		}
		return nil
	}
}

func TestBy(t *testing.T) {
	abcRule := By(func(_ context.Context, value interface{}) error {
		s, _ := value.(string)
		if s != "abc" {
			return errors.New("must be abc")
		}
		return nil
	})
	assert.Nil(t, ValidateWithContext(nil, "abc", abcRule))
	err := ValidateWithContext(nil, "xyz", abcRule)
	if assert.NotNil(t, err) {
		assert.Equal(t, "must be abc", err.Error())
	}

	xyzRule := By(stringEqual("xyz"))
	assert.Nil(t, ValidateWithContext(nil, "xyz", xyzRule))
	assert.NotNil(t, ValidateWithContext(nil, "abc", xyzRule))
}

type key int

func TestByWithContext(t *testing.T) {
	k := key(1)
	abcRule := By(func(ctx context.Context, value interface{}) error {
		if ctx.Value(k) != value.(string) {
			return errors.New("must be abc")
		}
		return nil
	})
	ctx := context.WithValue(context.Background(), k, "abc")
	assert.Nil(t, ValidateWithContext(ctx, "abc", abcRule))
	err := ValidateWithContext(ctx, "xyz", abcRule)
	if assert.NotNil(t, err) {
		assert.Equal(t, "must be abc", err.Error())
	}

	assert.NotNil(t, ValidateWithContext(nil, "abc", abcRule))
}

func Test_skipRule_Validate(t *testing.T) {
	assert.Nil(t, Skip.Validate(nil, 100))
}

func assertError(t *testing.T, expected string, err error, tag string) {
	if expected == "" {
		assert.NoError(t, err, tag)
	} else {
		assert.EqualError(t, err, expected, tag)
	}
}

type validateAbc struct{}

func (v *validateAbc) Validate(_ context.Context, obj interface{}) error {
	if !strings.Contains(obj.(string), "abc") {
		return errors.New("error abc")
	}
	return nil
}

type validateContextAbc struct{}

func (v *validateContextAbc) Validate(_ context.Context, obj interface{}) error {
	if !strings.Contains(obj.(string), "abc") {
		return errors.New("error abc")
	}
	return nil
}

type validateXyz struct{}

func (v *validateXyz) Validate(_ context.Context, obj interface{}) error {
	if !strings.Contains(obj.(string), "xyz") {
		return errors.New("error xyz")
	}
	return nil
}

type validateContextXyz struct{}

func (v *validateContextXyz) Validate(_ context.Context, obj interface{}) error {
	if !strings.Contains(obj.(string), "xyz") {
		return errors.New("error xyz")
	}
	return nil
}

type validateInternalError struct{}

func (v *validateInternalError) Validate(_ context.Context, obj interface{}) error {
	if strings.Contains(obj.(string), "internal") {
		return NewInternalError(errors.New("error internal"))
	}
	return nil
}

type Model1 struct {
	A string
	B string
	c string
	D *string
	E String123
	F *String123
	G string `json:"g"`
	H []string
	I map[string]string
}

type String123 string

func (s String123) Validate(_ context.Context) error {
	if !strings.Contains(string(s), "123") {
		return errors.New("error 123")
	}
	return nil
}

type Model2 struct {
	Model3
	M3   Model3
	M3AP []*Model3
	M4AP []*Model4
	B    string
}

type Model3 struct {
	A string
}

func (m Model3) Validate(ctx context.Context) error {
	return ValidateStructWithContext(ctx, &m,
		Field(&m.A, &validateAbc{}),
	)
}

type Model4 struct {
	A string
}

func (m Model4) Validate(ctx context.Context) error {
	return ValidateStructWithContext(ctx, &m,
		Field(&m.A, &validateContextAbc{}),
	)
}

type Model5 struct {
	Model4
	M4 Model4
	B  string
}

type StringValidate string

func (s StringValidate) Validate() error {
	return errors.New("called validate")
}

type StringValidateContext string

func (s StringValidateContext) Validate() error {
	if string(s) != "abc" {
		return errors.New("must be abc")
	}
	return nil
}

func (s StringValidateContext) ValidateWithContext(context.Context) error {
	if string(s) != "abc" {
		return errors.New("must be abc with context")
	}
	return nil
}

// ValidatableItemPtr is a test type that implements Validatable
type ValidatableItemPtr struct {
	Value string
}

func (v *ValidatableItemPtr) Validate(ctx context.Context) error {
	if v == nil {
		return nil
	}
	return ValidateWithContext(ctx, v.Value, Required)
}

type ValidatableString123 string

func (v ValidatableString123) Validate(ctx context.Context) error {
	if !strings.Contains(string(v), "123") {
		return errors.New("error 123")
	}
	return nil
}

func TestValidateSliceWithNilPointers(t *testing.T) {
	// Test Validatable slice containing nil pointers
	items := []*ValidatableItemPtr{
		{Value: "valid"},
		nil, // nil pointer
		{Value: "also valid"},
	}

	err := ValidateWithContext(nil, items)
	assert.Nil(t, err, "nil pointers in validatable slice should be handled")

	// Test: slice containing empty values
	itemsWithEmpty := []*ValidatableItemPtr{
		{Value: "valid"},
		{Value: ""}, // empty value
	}

	err = ValidateWithContext(nil, itemsWithEmpty)
	if assert.NotNil(t, err) {
		errs := err.(Errors)
		assert.Contains(t, errs, "1")
	}
}

func TestValidateSliceWithPointerToValidatable(t *testing.T) {
	// Test slice of pointers to Validatable
	slice := []*String123{
		func() *String123 { s := String123("123"); return &s }(),
		nil, // nil pointer - should be skipped
		func() *String123 { s := String123("abc"); return &s }(),
	}

	err := ValidateWithContext(nil, slice)
	if assert.NotNil(t, err) {
		errs := err.(Errors)
		assert.Contains(t, errs, "2")
		assert.NotContains(t, errs, "0")
		assert.NotContains(t, errs, "1") // nil should be skipped
	}
}

func TestValidateMapWithNilValues(t *testing.T) {
	// Test map values
	mp := map[string]ValidatableString123{
		"a": "123",
		"c": "abc",
	}

	err := ValidateWithContext(nil, mp)
	if assert.NotNil(t, err) {
		errs := err.(Errors)
		assert.Contains(t, errs, "c")
		assert.NotContains(t, errs, "a")
	}
}

func TestValidateNestedPointers(t *testing.T) {
	// Test nested pointers
	s := String123("abc")
	ptr1 := &s
	ptr2 := &ptr1

	err := ValidateWithContext(nil, ptr2)
	if assert.NotNil(t, err) {
		assert.Equal(t, "error 123", err.Error())
	}

	s2 := String123("123")
	ptr3 := &s2
	ptr4 := &ptr3

	err = ValidateWithContext(nil, ptr4)
	assert.Nil(t, err)
}

func TestValidateWithNilContext(t *testing.T) {
	// Ensure nil context is properly handled as Background
	err := ValidateWithContext(nil, "123", Required)
	assert.Nil(t, err)

	err = ValidateWithContext(nil, "", Required)
	assert.NotNil(t, err)
}

func TestValidateArrayOfValidatable(t *testing.T) {
	// Test fixed-size array
	arr := [2]String123{"123", "abc"}
	err := ValidateWithContext(nil, arr)
	if assert.NotNil(t, err) {
		errs := err.(Errors)
		assert.Contains(t, errs, "1")
		assert.NotContains(t, errs, "0")
	}
}

func TestValidateInterfaceValue(t *testing.T) {
	// Test interface{} type value
	var val interface{} = String123("123")
	err := ValidateWithContext(nil, val)
	assert.Nil(t, err)

	var val2 interface{} = String123("abc")
	err = ValidateWithContext(nil, val2)
	assert.NotNil(t, err)

	// Test nil interface
	var val3 interface{} = nil
	err = ValidateWithContext(nil, val3)
	assert.Nil(t, err)
}

func TestValidateEmptyCollections(t *testing.T) {
	// Test empty collections
	emptySlice := []String123{}
	err := ValidateWithContext(nil, emptySlice)
	assert.Nil(t, err)

	emptyMap := map[string]String123{}
	err = ValidateWithContext(nil, emptyMap)
	assert.Nil(t, err)

	emptyArray := [0]String123{}
	err = ValidateWithContext(nil, emptyArray)
	assert.Nil(t, err)
}
