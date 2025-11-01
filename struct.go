// Copyright 2016 Qiang Xue. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package validation

import (
	"context"
	"errors"
	"reflect"
	"strings"
)

// ErrStructPointer is the error that a struct being validated is not specified as a pointer.
var ErrStructPointer = errors.New("only a pointer to a struct can be validated")

// ValidateStructWithContext validates a struct with the given context.
// The only difference between ValidateStructWithContext and ValidateStruct is that the former will
// validate struct fields with the provided context.
// Please refer to ValidateStruct for the detailed instructions on how to use this function.
func ValidateStructWithContext(ctx context.Context, structPtr interface{}, fields ...FieldRules) error {
	if ctx == nil {
		ctx = context.Background()
	}

	value := reflect.ValueOf(structPtr)
	if value.Kind() != reflect.Ptr || !value.IsNil() && value.Elem().Kind() != reflect.Struct {
		// must be a pointer to a struct
		return NewInternalError(ErrStructPointer)
	}
	if value.IsNil() {
		// treat a nil struct pointer as valid
		return nil
	}
	value = value.Elem()

	errs := Errors{}

	for i, fr := range fields {
		ft, validateValue, err := fr.FindStructField(value, i)
		if err == ErrSkipFieldNotFound {
			continue
		} else if err != nil {
			return err
		}

		if err := ValidateWithContext(ctx, validateValue, fr.Rules()...); err != nil {
			if ie, ok := err.(InternalError); ok && ie.InternalError() != nil {
				return err
			}
			if ft.Anonymous {
				// merge errors from anonymous struct field
				if es, ok := err.(Errors); ok {
					for name, value := range es {
						errs[name] = value
					}
					continue
				}
			}
			errs[getOpts(ctx).getErrorFieldNameFunc(ft)] = err
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// ErrorFieldName returns the name resolved from tagName for the provided struct field pointer.
func ErrorFieldName(structPtr interface{}, fieldPtr interface{}, tagName string) (string, error) {
	value := reflect.ValueOf(structPtr)
	if value.Kind() != reflect.Ptr || !value.IsNil() && value.Elem().Kind() != reflect.Struct {
		// must be a pointer to a struct
		return "", NewInternalError(ErrStructPointer)
	}
	if value.IsNil() {
		// treat a nil struct pointer as valid
		return "", nil
	}
	value = value.Elem()

	fv := reflect.ValueOf(fieldPtr)
	if fv.Kind() != reflect.Ptr {
		// must be a pointer to a field
		return "", NewInternalError(ErrFieldPointer(0))
	}
	ft := findStructField(value, fv)
	if ft == nil {
		return "", NewInternalError(ErrFieldNotFound(0))
	}
	return getErrorFieldName(ft, tagName), nil
}

// getErrorFieldName returns the name that should be used to represent the validation error of a struct field.
func getErrorFieldName(f *reflect.StructField, tagName string) string {
	if tag := f.Tag.Get(tagName); tag != "" && tag != "-" {
		if cps := strings.SplitN(tag, ",", 2); cps[0] != "" {
			return cps[0]
		}
	}
	return f.Name
}

func DefaultGetErrorFieldName(f *reflect.StructField) string {
	return getErrorFieldName(f, "json")
}
