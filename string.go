// Copyright 2016 Qiang Xue. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package validation

import "context"

var _ Rule = (*StringRule)(nil)

type stringValidator func(string) bool

type stringValidatorWithContext func(context.Context, string) bool

// StringRule is a rule that checks a string variable using a specified stringValidator.
type StringRule struct {
	validate stringValidatorWithContext
	err      Error
}

// NewStringRule creates a new validation rule using a function that takes a string value and returns a bool.
// The rule returned will use the function to check if a given string or byte slice is valid or not.
// An empty value is considered to be valid. Please use the Required rule to make sure a value is not empty.
func NewStringRule(validator stringValidator, message string) StringRule {
	return StringRule{
		validate: func(_ context.Context, s string) bool { return validator(s) },
		err:      NewError("", message),
	}
}

// NewStringRuleWithError creates a new validation rule using a function that takes a string value and returns a bool.
// The rule returned will use the function to check if a given string or byte slice is valid or not.
// An empty value is considered to be valid. Please use the Required rule to make sure a value is not empty.
func NewStringRuleWithError(validator stringValidator, err Error) StringRule {
	return StringRule{
		validate: func(_ context.Context, s string) bool { return validator(s) },
		err:      err,
	}
}

// NewStringRuleWithContext creates a new validation rule using a function that takes a context and a string value and returns a bool.
// The rule returned will use the function to check if a given string or byte slice is valid or not.
// An empty value is considered to be valid. Please use the Required rule to make sure a value is not empty.
func NewStringRuleWithContext(validator stringValidatorWithContext, message string) StringRule {
	return StringRule{
		validate: validator,
		err:      NewError("", message),
	}
}

// NewStringRuleWithContextError creates a new validation rule using a function that takes a context and a string value and returns a bool.
// The rule returned will use the function to check if a given string or byte slice is valid or not.
// An empty value is considered to be valid. Please use the Required rule to make sure a value is not empty.
func NewStringRuleWithContextError(validator stringValidatorWithContext, message string) StringRule {
	return StringRule{
		validate: validator,
		err:      NewError("", message),
	}
}

// Error sets the error message for the rule.
func (r StringRule) Error(message string) StringRule {
	r.err = r.err.SetMessage(message)
	return r
}

// ErrorObject sets the error struct for the rule.
func (r StringRule) ErrorObject(err Error) StringRule {
	r.err = err
	return r
}

// Validate checks if the given value is valid or not.
func (r StringRule) Validate(ctx context.Context, value interface{}) error {
	if ctx == nil {
		ctx = context.Background()
	}

	value, isNil := indirectWithOptions(value, GetOptions(ctx))
	if isNil || IsEmpty(value) {
		return nil
	}

	str, err := EnsureString(value)
	if err != nil {
		return err
	}

	if r.validate(ctx, str) {
		return nil
	}

	return r.err
}
