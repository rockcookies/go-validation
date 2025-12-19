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

var (
	// ErrSkipFieldNotFound is returned when a field is not found but validation should be skipped.
	ErrSkipFieldNotFound = errors.New("field not found, skipping validation")

	// ErrFieldRequired indicates that a required field is missing.
	ErrFieldRequired = NewError("validation_field_required", "missing required field: {{.field_name}}")
)

type (

	// ErrFieldPointer is the error that a field is not specified as a pointer.
	ErrFieldPointer int

	// ErrFieldNotFound is the error that a field cannot be found in the struct.
	ErrFieldNotFound int

	// FieldRules represents a rule set associated with a struct field.
	FieldRules interface {
		Rules() []Rule
		FindStructField(structValue reflect.Value, idx int) (*reflect.StructField, any, error)
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

// NamedFieldRules represents a rule set associated with a named struct field.
type NamedFieldRules struct {
	name             string
	rules            []Rule
	validatePtrValue bool
	skipIfNotFound   bool
}

var _ FieldRules = (*NamedFieldRules)(nil)

func (n *NamedFieldRules) Name() string {
	return n.name
}

func (n *NamedFieldRules) Rules() []Rule {
	return n.rules
}

func (n *NamedFieldRules) SkipIfNotFound() bool {
	return n.skipIfNotFound
}

func (n *NamedFieldRules) SetSkipIfNotFound(skip bool) *NamedFieldRules {
	n.skipIfNotFound = skip
	return n
}

// toFieldName converts a field name to its struct field representation.
// If the name starts with a lowercase letter, it converts the first letter to uppercase.
func toFieldName(name string) string {
	if name[0] >= 'a' && name[0] <= 'z' {
		return strings.ToUpper(name[:1]) + name[1:]
	}
	return name
}

func (n *NamedFieldRules) FindStructField(structValue reflect.Value, idx int) (*reflect.StructField, any, error) {
	name := toFieldName(n.name)

	var ft *reflect.StructField

	fv := structValue.FieldByName(name)
	if fv.IsValid() {
		sf, ok := structValue.Type().FieldByName(name)
		if ok {
			ft = &sf
			fv = structValue.FieldByName(name).Addr()
		}
	}

	if ft == nil {
		if n.skipIfNotFound {
			return nil, nil, ErrSkipFieldNotFound
		}

		return nil, nil, ErrFieldRequired.SetParams(map[string]any{"field_name": n.name})
	}

	var value interface{}
	if !n.validatePtrValue {
		value = fv.Elem().Interface()
	} else {
		value = fv.Interface()
	}

	return ft, value, nil
}

// NamedField specifies a named field and the corresponding validation rules.
func NamedField(name string, rules ...Rule) *NamedFieldRules {
	return &NamedFieldRules{
		name:  name,
		rules: rules,
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
func NamedStructField(name string, fields ...FieldRules) *NamedFieldRules {
	return &NamedFieldRules{
		name: name,
		rules: []Rule{&inlineRule{
			f: func(ctx context.Context, value interface{}) error {
				return ValidateStructWithContext(ctx, value, fields...)
			},
		}},
		validatePtrValue: true,
	}
}

type PointerFieldRules struct {
	fieldPtr         interface{}
	rules            []Rule
	validatePtrValue bool
}

var _ FieldRules = (*PointerFieldRules)(nil)

func (f *PointerFieldRules) Rules() []Rule {
	return f.rules
}

func (f *PointerFieldRules) FindStructField(structValue reflect.Value, idx int) (*reflect.StructField, any, error) {
	fv := reflect.ValueOf(f.fieldPtr)
	if fv.Kind() != reflect.Ptr {
		return nil, nil, NewInternalError(ErrFieldPointer(idx))
	}

	ft := findStructField(structValue, fv)
	if ft == nil {
		return nil, nil, NewInternalError(ErrFieldNotFound(idx))
	}

	var value interface{}
	if !f.validatePtrValue {
		value = fv.Elem().Interface()
	} else {
		value = fv.Interface()
	}

	return ft, value, nil
}

// Field specifies a struct field and the corresponding validation rules.
// The struct field must be specified as a pointer to it.
func Field(fieldPtr interface{}, rules ...Rule) FieldRules {
	return &PointerFieldRules{
		fieldPtr: fieldPtr,
		rules:    rules,
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
func FieldStruct(structPtr interface{}, fields ...FieldRules) *PointerFieldRules {
	return &PointerFieldRules{
		fieldPtr: structPtr,
		rules: []Rule{&inlineRule{
			f: func(ctx context.Context, value interface{}) error {
				return ValidateStructWithContext(ctx, value, fields...)
			},
		}},
		validatePtrValue: true,
	}
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
