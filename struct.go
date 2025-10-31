// Copyright 2016 Qiang Xue. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package validation

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// ErrStructPointer is the error that a struct being validated is not specified as a pointer.
var ErrStructPointer = errors.New("only a pointer to a struct can be validated")

type (
	// ErrFieldPointer is the error that a field is not specified as a pointer.
	ErrFieldPointer int

	// ErrFieldNotFound is the error that a field cannot be found in the struct.
	ErrFieldNotFound int

	// FieldRules represents a rule set associated with a struct field.
	FieldRules struct {
		name             string
		isNamedField     bool
		fieldPtr         interface{}
		rules            []Rule
		validatePtrValue bool
	}
)

// Error returns the error string of ErrFieldPointer.
func (e ErrFieldPointer) Error() string {
	return fmt.Sprintf("field #%v must be specified as a pointer", int(e))
}

// Error returns the error string of ErrFieldNotFound.
func (e ErrFieldNotFound) Error() string {
	return fmt.Sprintf("field #%v cannot be found in the struct", int(e))
}

// ValidateStructWithContext validates a struct with the given context.
// The only difference between ValidateStructWithContext and ValidateStruct is that the former will
// validate struct fields with the provided context.
// Please refer to ValidateStruct for the detailed instructions on how to use this function.
func ValidateStructWithContext(ctx context.Context, structPtr interface{}, fields ...*FieldRules) error {
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
		var fv reflect.Value
		var ft *reflect.StructField

		if fr.isNamedField {
			var ok bool
			fv, ft, ok = getOpts(ctx).findStructFieldByNameFunc(value, fr.name)
			if !ok {
				return NewInternalError(ErrFieldNotFound(i))
			}

			if fv.Kind() != reflect.Ptr {
				if fv.CanAddr() {
					fv = fv.Addr()
				} else {
					return NewInternalError(ErrFieldPointer(i))
				}
			}
		} else {
			fv = reflect.ValueOf(fr.fieldPtr)
			if fv.Kind() != reflect.Ptr {
				return NewInternalError(ErrFieldPointer(i))
			}

			ft = findStructField(value, fv)
			if ft == nil {
				return NewInternalError(ErrFieldNotFound(i))
			}
		}

		var validateValue interface{}
		if !fr.validatePtrValue {
			validateValue = fv.Elem().Interface()
		} else {
			validateValue = fv.Interface()
		}

		if err := ValidateWithContext(ctx, validateValue, fr.rules...); err != nil {
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

// NamedField specifies a named field and the corresponding validation rules.
func NamedField(name string, rules ...Rule) *FieldRules {
	return &FieldRules{
		name:         name,
		isNamedField: true,
		rules:        rules,
	}
}

// Field specifies a struct field and the corresponding validation rules.
// The struct field must be specified as a pointer to it.
func Field(fieldPtr interface{}, rules ...Rule) *FieldRules {
	return &FieldRules{
		fieldPtr: fieldPtr,
		rules:    rules,
	}
}

// NamedStructField specifies a named struct field and the corresponding validation field rules.
// example,
//
//	value := struct {
//		NestedStruct struct {
//		  Name string
//	  }
//	}{NestedStruct: struct{Name string}{}}
//	err := validation.ValidateStruct(
//	  &value,
//		validation.NamedStructField(
//	    "nestedStruct",
//	    validation.Field(&value.NestedStruct.Name, validation.Required),
//	  ),
//	)
func NamedStructField(name string, fields ...*FieldRules) *FieldRules {
	return &FieldRules{
		name:         name,
		isNamedField: true,
		rules: []Rule{&inlineRule{
			f: func(ctx context.Context, value interface{}) error {
				return ValidateStructWithContext(ctx, value, fields...)
			},
		}},
		validatePtrValue: true,
	}
}

// FieldStruct specifies a struct field and the corresponding validation field rules.
// The struct field must be specified as a pointer to struct.
// example,
//
//	value := struct {
//		NestedStruct struct {
//		  Name string
//	  }
//	}{NestedStruct: struct{Name string}{}}
//	err := validation.ValidateStruct(
//	  &value,
//		validation.FieldStruct(
//	    &value.NestedStruct,
//	    validation.Field(&value.NestedStruct.Name, validation.Required),
//	  ),
//	)
func FieldStruct(structPtr interface{}, fields ...*FieldRules) *FieldRules {
	return &FieldRules{
		fieldPtr: structPtr,
		rules: []Rule{&inlineRule{
			f: func(ctx context.Context, value interface{}) error {
				return ValidateStructWithContext(ctx, value, fields...)
			},
		}},
		validatePtrValue: true,
	}
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

func DefaultFindStructFieldByName(structValue reflect.Value, name string) (reflect.Value, *reflect.StructField, bool) {
	if len(name) == 0 {
		return reflect.Value{}, nil, false
	}

	if name[0] >= 'a' && name[0] <= 'z' {
		name = strings.ToUpper(name[:1]) + name[1:]
	}

	fv := structValue.FieldByName(name)
	if fv.IsValid() {
		ft, _ := structValue.Type().FieldByName(name)
		return fv, &ft, true
	}

	return reflect.Value{}, nil, false
}

// findStructField looks for a field in the given struct.
// The field being looked for should be a pointer to the actual struct field.
// If found, the field info will be returned. Otherwise, nil will be returned.
func findStructField(structValue reflect.Value, fieldValue reflect.Value) *reflect.StructField {
	ptr := fieldValue.Pointer()
	for i := structValue.NumField() - 1; i >= 0; i-- {
		sf := structValue.Type().Field(i)
		if ptr == structValue.Field(i).UnsafeAddr() {
			// do additional type comparison because it's possible that the address of
			// an embedded struct is the same as the first field of the embedded struct
			if sf.Type == fieldValue.Elem().Type() {
				return &sf
			}
		}
		if sf.Anonymous {
			// delve into anonymous struct to look for the field
			fi := structValue.Field(i)
			if sf.Type.Kind() == reflect.Ptr {
				fi = fi.Elem()
			}
			if fi.Kind() == reflect.Struct {
				if f := findStructField(fi, fieldValue); f != nil {
					return f
				}
			}
		}
	}
	return nil
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
