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

// ValidateStruct validates a struct.
// The structPtr parameter must be a pointer to a struct. If structPtr is nil, it is considered valid.
// The fields parameter specifies which struct fields to be validated and the validation rules for each field.
// Each element in fields corresponds to one struct field. The order of the elements in fields does not
// have to be the same as the order of the struct fields.
//
// For each element in fields, if the specified struct field is found, its value will be validated
// against the validation rules associated with that field. If the field is not found, it will be skipped.
// If the field is an anonymous struct field and there are validation errors for that field,
// the validation errors will be merged into the top-level validation errors.
//
// If there are validation errors, they will be returned as an Errors object,
// where each key is the name of a struct field and the corresponding value is the validation error for that field.
// The name of a struct field is determined by the "json" tag of the field. If the "json" tag is not present,
// the actual field name will be used.
//
// Example:
//
//	type User struct {
//	    ID    int
//	    Name  string `json:"name"`
//	    Email string
//	}
//
//	user := &User{ID: 1, Name: "", Email: "invalid-email"}
//	err := validation.ValidateStruct(user,
//	    validation.Field(&user.Name, validation.Required),
//	    validation.Field(&user.Email, validation.Required, validation.Email),
//	)
//	if err != nil {
//	    // err is of type validation.Errors
//	    fmt.Println(err)
//	    // Output:
//	    // name: cannot be blank; Email: must be a valid email address.
//	}
func ValidateStruct(structPtr interface{}, fields ...FieldRules) error {
	return ValidateStructWithContext(context.Background(), structPtr, fields...)
}

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
