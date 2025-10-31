// Copyright 2016 Qiang Xue. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package validation

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func validateMe(s string) bool {
	return s == "me"
}

func TestNewStringRule(t *testing.T) {
	v := NewStringRule(validateMe, "abc")

	assert.NotNil(t, v.validate)
	assert.Equal(t, "", v.err.Code())
	assert.Equal(t, "abc", v.err.Message())
}

func TestNewStringRuleWithError(t *testing.T) {
	err := NewError("C", "abc")
	v := NewStringRuleWithError(validateMe, err)

	assert.NotNil(t, v.validate)
	assert.Equal(t, err, v.err)
	assert.Equal(t, "C", v.err.Code())
	assert.Equal(t, "abc", v.err.Message())
}

func TestStringRule_Error(t *testing.T) {
	err := NewError("code", "abc")
	v := NewStringRuleWithError(validateMe, err).Error("abc")
	assert.Equal(t, "code", v.err.Code())
	assert.Equal(t, "abc", v.err.Message())

	v2 := v.Error("correct")
	assert.Equal(t, "code", v.err.Code())
	assert.Equal(t, "correct", v2.err.Message())
	assert.Equal(t, "abc", v.err.Message())
}

func TestStringValidator_Validate(t *testing.T) {
	v := NewStringRule(validateMe, "wrong_rule").Error("wrong")

	value := "me"

	err := v.Validate(nil, value)
	assert.Nil(t, err)

	err = v.Validate(nil, &value)
	assert.Nil(t, err)

	value = ""

	err = v.Validate(nil, value)
	assert.Nil(t, err)

	err = v.Validate(nil, &value)
	assert.Nil(t, err)

	nullValue := sql.NullString{String: "me", Valid: true}
	err = v.Validate(nil, nullValue)
	assert.Nil(t, err)

	nullValue = sql.NullString{String: "", Valid: true}
	err = v.Validate(nil, nullValue)
	assert.Nil(t, err)

	var s *string
	err = v.Validate(nil, s)
	assert.Nil(t, err)

	err = v.Validate(nil, "not me")
	if assert.NotNil(t, err) {
		assert.Equal(t, "wrong", err.Error())
	}

	err = v.Validate(nil, 100)
	if assert.NotNil(t, err) {
		assert.NotEqual(t, "wrong", err.Error())
	}

	v2 := v.Error("Wrong!")
	err = v2.Validate(nil, "not me")
	if assert.NotNil(t, err) {
		assert.Equal(t, "Wrong!", err.Error())
	}
}

func TestGetErrorFieldName(t *testing.T) {
	type A struct {
		T0 string
		T1 string `json:"t1"`
		T2 string `json:"t2,omitempty"`
		T3 string `json:",omitempty"`
		T4 string `json:"t4,x1,omitempty"`
	}
	tests := []struct {
		tag   string
		field string
		name  string
	}{
		{"t1", "T0", "T0"},
		{"t2", "T1", "t1"},
		{"t3", "T2", "t2"},
		{"t4", "T3", "T3"},
		{"t5", "T4", "t4"},
	}
	a := reflect.TypeOf(A{})
	for _, test := range tests {
		field, _ := a.FieldByName(test.field)
		assert.Equal(t, test.name, getErrorFieldName(&field, "json"), test.tag)
	}
}

func TestStringRule_ErrorObject(t *testing.T) {
	r := NewStringRule(validateMe, "wrong_rule")

	err := NewError("code", "abc")
	r = r.ErrorObject(err)

	assert.Equal(t, err, r.err)
	assert.Equal(t, "code", r.err.Code())
	assert.Equal(t, "abc", r.err.Message())
}

func TestNewStringRuleWithContext(t *testing.T) {
	type ctxKey string
	key := ctxKey("expected_value")

	// Create a validation rule that uses context
	rule := NewStringRuleWithContext(
		func(ctx context.Context, s string) bool {
			expected := ctx.Value(key)
			if expected == nil {
				return false
			}
			return s == expected.(string)
		},
		"value does not match expected")

	// Test: value matches context value
	ctx1 := context.WithValue(context.Background(), key, "expected_value")
	err := rule.Validate(ctx1, "expected_value")
	assert.Nil(t, err)

	// Test: value does not match context value
	err = rule.Validate(ctx1, "wrong_value")
	if assert.NotNil(t, err) {
		assert.Equal(t, "value does not match expected", err.Error())
	}

	// Test: different context
	ctx2 := context.WithValue(context.Background(), key, "another_value")
	err = rule.Validate(ctx2, "another_value")
	assert.Nil(t, err)

	// Test: nil context
	err = rule.Validate(nil, "anything")
	assert.NotNil(t, err)

	// Test: empty string (should skip validation)
	err = rule.Validate(ctx1, "")
	assert.Nil(t, err)
}

func TestNewStringRuleWithContextError(t *testing.T) {
	type ctxKey string
	key := ctxKey("min_length")

	customErr := NewError("validation_min_length", "string is too short")

	// Create a context validation rule with custom error
	rule := NewStringRuleWithContextError(
		func(ctx context.Context, s string) bool {
			minLen := ctx.Value(key)
			if minLen == nil {
				return true
			}
			return len(s) >= minLen.(int)
		},
		"string is too short")

	ctx := context.WithValue(context.Background(), key, 5)

	// Test: length is sufficient
	err := rule.Validate(ctx, "hello world")
	assert.Nil(t, err)

	// Test: length is insufficient
	err = rule.Validate(ctx, "hi")
	if assert.NotNil(t, err) {
		assert.Equal(t, "string is too short", err.Error())
	}

	// Test: custom error message
	ruleWithCustomErr := rule.ErrorObject(customErr)
	err = ruleWithCustomErr.Validate(ctx, "hi")
	if assert.NotNil(t, err) {
		assert.Equal(t, "validation_min_length", err.(Error).Code())
		assert.Equal(t, "string is too short", err.Error())
	}
}

func TestStringRuleWithContextAndOptions(t *testing.T) {
	type ctxKey string
	key := ctxKey("allowed_prefix")

	// Create validation rule using context and options
	rule := NewStringRuleWithContext(
		func(ctx context.Context, s string) bool {
			prefix := ctx.Value(key)
			if prefix == nil {
				return false
			}
			return len(s) > 0 && s[0:1] == prefix.(string)
		},
		"string must start with allowed prefix")

	// Custom ValuerFunc to handle special types
	type CustomString struct {
		Value string
	}

	customValuer := func(v any) (any, bool) {
		if cs, ok := v.(CustomString); ok {
			return cs.Value, true
		}
		return v, false
	}

	ctx := context.WithValue(context.Background(), key, "A")
	ctx = WithOptions(ctx, WithValuerFunc(customValuer))

	// Test: regular string
	err := rule.Validate(ctx, "Apple")
	assert.Nil(t, err)

	err = rule.Validate(ctx, "Banana")
	assert.NotNil(t, err)

	// Test: custom type (converted through ValuerFunc)
	err = rule.Validate(ctx, CustomString{Value: "Amazing"})
	assert.Nil(t, err)

	err = rule.Validate(ctx, CustomString{Value: "Bad"})
	assert.NotNil(t, err)
}

func TestStringRuleWithContextCancellation(t *testing.T) {
	// Test context cancellation scenario
	rule := NewStringRuleWithContext(
		func(ctx context.Context, s string) bool {
			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return false
			default:
				return s == "valid"
			}
		},
		"validation failed")

	// Test: normal context
	ctx1 := context.Background()
	err := rule.Validate(ctx1, "valid")
	assert.Nil(t, err)

	// Test: cancelled context
	ctx2, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	err = rule.Validate(ctx2, "valid")
	assert.NotNil(t, err)
}

func TestNewStringRuleWithErrorCustom(t *testing.T) {
	customErr := NewError("custom_code", "custom message")

	validator := func(s string) bool {
		return len(s) >= 3
	}

	rule := NewStringRuleWithError(validator, customErr)

	// Test: validation passes
	err := rule.Validate(nil, "hello")
	assert.Nil(t, err)

	// Test: validation fails, check custom error
	err = rule.Validate(nil, "ab")
	if assert.NotNil(t, err) {
		errObj, ok := err.(Error)
		assert.True(t, ok)
		assert.Equal(t, "custom_code", errObj.Code())
		assert.Equal(t, "custom message", errObj.Message())
	}

	// Test: modify error message
	rule2 := rule.Error("modified message")
	err = rule2.Validate(nil, "ab")
	if assert.NotNil(t, err) {
		errObj, ok := err.(Error)
		assert.True(t, ok)
		assert.Equal(t, "custom_code", errObj.Code())
		assert.Equal(t, "modified message", errObj.Message())
	}

	// Ensure original rule is not modified
	err = rule.Validate(nil, "ab")
	if assert.NotNil(t, err) {
		assert.Equal(t, "custom message", err.Error())
	}
}
