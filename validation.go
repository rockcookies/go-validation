// Copyright 2016 Qiang Xue. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package validation provides configurable and extensible rules for validating data of various types.
package validation

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
)

type (
	// Validatable is the interface indicating the type implementing it supports context-aware data validation.
	Validatable interface {
		// Validate validates the data with the given context and returns an error if validation fails.
		Validate(ctx context.Context) error
	}

	Rule interface {
		Validate(ctx context.Context, value interface{}) error
	}

	// RuleFunc represents a validator function that is context-aware.
	// You may wrap it as a Rule by calling WithContext().
	RuleFunc func(ctx context.Context, value interface{}) error
)

var (
	// Skip is a special validation rule that indicates all rules following it should be skipped.
	Skip = skipRule{skip: true}

	validatableType = reflect.TypeOf((*Validatable)(nil)).Elem()
)

// Validate validates the given value and returns the validation error, if any.
// Validate performs validation using the following steps:
//  1. For each rule, call its Validate() to validate the value.
//  2. If the value being validated implements Validatable, call the value's Validate()
//     and return with the validation result.
//  3. If the value being validated is a map/slice/array, and the element type implements Validatable,
//     for each element call the element value's Validate(). Return with the validation result.
//
// Validate is equivalent to calling ValidateWithContext with a nil context.
func Validate(value interface{}, rules ...Rule) error {
	return ValidateWithContext(context.Background(), value, rules...)
}

// ValidateWithContext validates the given value with the given context and returns the validation error, if any.
//
// ValidateWithContext performs validation using the following steps:
//  1. For each rule, call its Validate() to validate the value.
//     Otherwise call `Validate()` of the rule. Return if any error is found.
//  2. If the value being validated implements `ValidatableWithContext`, call the value's `ValidateWithContext()`
//     and return with the validation result.
//  3. If the value being validated implements `Validatable`, call the value's `Validate()`
//     and return with the validation result.
//  4. If the value being validated is a map/slice/array, and the element type implements `ValidatableWithContext`,
//     for each element call the element value's `ValidateWithContext()`. Return with the validation result.
//  5. If the value being validated is a map/slice/array, and the element type implements `Validatable`,
//     for each element call the element value's `Validate()`. Return with the validation result.
func ValidateWithContext(ctx context.Context, value interface{}, rules ...Rule) error {
	if ctx == nil {
		ctx = context.Background()
	}

	for _, rule := range rules {
		if s, ok := rule.(skipRule); ok && s.skip {
			return nil
		}

		if err := rule.Validate(ctx, value); err != nil {
			return err
		}
	}

	rv := reflect.ValueOf(value)
	if (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface) && rv.IsNil() {
		return nil
	}

	if v, ok := value.(Validatable); ok {
		return v.Validate(ctx)
	}

	switch rv.Kind() {
	case reflect.Map:
		if rv.Type().Elem().Implements(validatableType) {
			return validateMap(ctx, rv)
		}
	case reflect.Slice, reflect.Array:
		if rv.Type().Elem().Implements(validatableType) {
			return validateSlice(ctx, rv)
		}
	case reflect.Ptr, reflect.Interface:
		return ValidateWithContext(ctx, rv.Elem().Interface())
	}

	return nil
}

// validateMap validates a map of validatable elements with the given context.
func validateMap(ctx context.Context, rv reflect.Value) error {
	errs := Errors{}
	for _, key := range rv.MapKeys() {
		if mv := rv.MapIndex(key).Interface(); mv != nil {
			if err := mv.(Validatable).Validate(ctx); err != nil {
				errs[fmt.Sprintf("%v", key.Interface())] = err
			}
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// validateSlice validates a slice/array of validatable elements with the given context.
func validateSlice(ctx context.Context, rv reflect.Value) error {
	errs := Errors{}
	l := rv.Len()
	for i := 0; i < l; i++ {
		v := rv.Index(i)
		if v.Kind() == reflect.Ptr && v.IsNil() {
			continue
		}
		if ev := v.Interface(); ev != nil {
			if err := ev.(Validatable).Validate(ctx); err != nil {
				errs[strconv.Itoa(i)] = err
			}
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

var _ Rule = (*skipRule)(nil)

type skipRule struct {
	skip bool
}

func (r skipRule) Validate(context.Context, interface{}) error {
	return nil
}

// When determines if all rules following it should be skipped.
func (r skipRule) When(condition bool) skipRule {
	r.skip = condition
	return r
}

type inlineRule struct {
	f RuleFunc
}

func (r *inlineRule) Validate(ctx context.Context, value interface{}) error {
	return r.f(ctx, value)
}

// By wraps a RuleFunc into a Rule.
func By(f RuleFunc) Rule {
	return &inlineRule{f: f}
}
