// Copyright 2016 Qiang Xue. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package validation

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrFieldPointer_Error(t *testing.T) {
	err := ErrFieldPointer(0)
	assert.Equal(t, "field #0 must be specified as a pointer", err.Error())

	err = ErrFieldPointer(5)
	assert.Equal(t, "field #5 must be specified as a pointer", err.Error())
}

func TestErrFieldNotFound_Error(t *testing.T) {
	err := ErrFieldNotFound(0)
	assert.Equal(t, "field #0 cannot be found in the struct", err.Error())

	err = ErrFieldNotFound(3)
	assert.Equal(t, "field #3 cannot be found in the struct", err.Error())
}

func TestNamedField(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		rules     []Rule
		wantName  string
		wantRules int
	}{
		{
			name:      "simple named field",
			fieldName: "Field1",
			rules:     []Rule{Required},
			wantName:  "Field1",
			wantRules: 1,
		},
		{
			name:      "named field with multiple rules",
			fieldName: "Email",
			rules:     []Rule{Required, Length(5, 100)},
			wantName:  "Email",
			wantRules: 2,
		},
		{
			name:      "named field with no rules",
			fieldName: "Optional",
			rules:     nil,
			wantName:  "Optional",
			wantRules: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fr := NamedField(tt.fieldName, tt.rules...)
			assert.Equal(t, tt.wantName, fr.Name())
			assert.Len(t, fr.Rules(), tt.wantRules)
			assert.False(t, fr.SkipIfNotFound())
			assert.False(t, fr.validatePtrValue)
		})
	}
}

func TestNamedFieldRules_SetSkipIfNotFound(t *testing.T) {
	fr := NamedField("Field1", Required)
	assert.False(t, fr.SkipIfNotFound())

	fr.SetSkipIfNotFound(true)
	assert.True(t, fr.SkipIfNotFound())

	fr.SetSkipIfNotFound(false)
	assert.False(t, fr.SkipIfNotFound())
}

func TestNamedFieldRules_FindStructField(t *testing.T) {
	type TestStruct struct {
		Name  string
		Email string
		Age   int
		score int // unexported
	}

	tests := []struct {
		name           string
		setupFunc      func() (*TestStruct, *NamedFieldRules)
		wantErr        bool
		wantFieldName  string
		wantValue      interface{}
		skipIfNotFound bool
	}{
		{
			name: "find capitalized field",
			setupFunc: func() (*TestStruct, *NamedFieldRules) {
				ts := &TestStruct{Name: "John", Email: "john@example.com", Age: 25}
				return ts, NamedField("Name", Required)
			},
			wantErr:       false,
			wantFieldName: "Name",
			wantValue:     "John",
		},
		{
			name: "find field with lowercase first letter - auto capitalize",
			setupFunc: func() (*TestStruct, *NamedFieldRules) {
				ts := &TestStruct{Name: "John", Email: "john@example.com", Age: 25}
				return ts, NamedField("email", Required)
			},
			wantErr:       false,
			wantFieldName: "Email",
			wantValue:     "john@example.com",
		},
		{
			name: "field not found - error",
			setupFunc: func() (*TestStruct, *NamedFieldRules) {
				ts := &TestStruct{Name: "John"}
				return ts, NamedField("NonExistent", Required)
			},
			wantErr: true,
		},
		{
			name: "field not found - skip if not found",
			setupFunc: func() (*TestStruct, *NamedFieldRules) {
				ts := &TestStruct{Name: "John"}
				fr := NamedField("NonExistent", Required)
				fr.SetSkipIfNotFound(true)
				return ts, fr
			},
			wantErr:        true,
			skipIfNotFound: true,
		},
		{
			name: "find integer field",
			setupFunc: func() (*TestStruct, *NamedFieldRules) {
				ts := &TestStruct{Name: "John", Age: 30}
				return ts, NamedField("Age", Required)
			},
			wantErr:       false,
			wantFieldName: "Age",
			wantValue:     30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, fr := tt.setupFunc()
			structValue := reflect.ValueOf(ts).Elem()

			ft, value, err := fr.FindStructField(structValue, 0)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.skipIfNotFound {
					assert.Equal(t, ErrSkipFieldNotFound, err)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, ft)
			assert.Equal(t, tt.wantFieldName, ft.Name)
			assert.Equal(t, tt.wantValue, value)
		})
	}
}

func TestNamedStructField(t *testing.T) {
	type Inner struct {
		Value string
	}
	type Outer struct {
		InnerStruct Inner
	}

	fr := NamedStructField("InnerStruct",
		NamedField("Value", Required),
	)

	assert.Equal(t, "InnerStruct", fr.Name())
	assert.Len(t, fr.Rules(), 1)
	assert.True(t, fr.validatePtrValue)

	// Test successful validation
	outer := &Outer{InnerStruct: Inner{Value: "test"}}
	structValue := reflect.ValueOf(outer).Elem()
	ft, value, err := fr.FindStructField(structValue, 0)
	assert.NoError(t, err)
	assert.NotNil(t, ft)
	assert.Equal(t, "InnerStruct", ft.Name)

	// The value should be a pointer to the struct
	innerPtr, ok := value.(*Inner)
	assert.True(t, ok)
	assert.Equal(t, "test", innerPtr.Value)
}

func TestField(t *testing.T) {
	type TestStruct struct {
		Name  string
		Email string
		Age   int
	}

	ts := &TestStruct{Name: "John", Email: "john@example.com", Age: 25}

	tests := []struct {
		name      string
		setupFunc func() FieldRules
		wantRules int
	}{
		{
			name: "field with single rule",
			setupFunc: func() FieldRules {
				return Field(&ts.Name, Required)
			},
			wantRules: 1,
		},
		{
			name: "field with multiple rules",
			setupFunc: func() FieldRules {
				return Field(&ts.Email, Required, Length(5, 100))
			},
			wantRules: 2,
		},
		{
			name: "field with no rules",
			setupFunc: func() FieldRules {
				return Field(&ts.Age)
			},
			wantRules: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fr := tt.setupFunc()
			assert.Len(t, fr.Rules(), tt.wantRules)
		})
	}
}

func TestPointerFieldRules_FindStructField(t *testing.T) {
	type TestStruct struct {
		Name  string
		Email string
		Age   int
	}

	tests := []struct {
		name          string
		setupFunc     func() (*TestStruct, FieldRules)
		wantErr       bool
		wantFieldName string
		wantValue     interface{}
		isInternal    bool
	}{
		{
			name: "find field successfully",
			setupFunc: func() (*TestStruct, FieldRules) {
				ts := &TestStruct{Name: "John", Email: "john@example.com", Age: 25}
				return ts, Field(&ts.Name, Required)
			},
			wantErr:       false,
			wantFieldName: "Name",
			wantValue:     "John",
		},
		{
			name: "find integer field",
			setupFunc: func() (*TestStruct, FieldRules) {
				ts := &TestStruct{Name: "John", Age: 30}
				return ts, Field(&ts.Age, Required)
			},
			wantErr:       false,
			wantFieldName: "Age",
			wantValue:     30,
		},
		{
			name: "field pointer from different struct - error",
			setupFunc: func() (*TestStruct, FieldRules) {
				ts := &TestStruct{Name: "John"}
				other := &TestStruct{Name: "Jane"}
				return ts, Field(&other.Name, Required)
			},
			wantErr:    true,
			isInternal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, fr := tt.setupFunc()
			structValue := reflect.ValueOf(ts).Elem()

			pfr, ok := fr.(*PointerFieldRules)
			assert.True(t, ok)

			ft, value, err := pfr.FindStructField(structValue, 0)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.isInternal {
					_, ok := err.(InternalError)
					assert.True(t, ok)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, ft)
			assert.Equal(t, tt.wantFieldName, ft.Name)
			assert.Equal(t, tt.wantValue, value)
		})
	}
}

func TestPointerFieldRules_FindStructField_NonPointer(t *testing.T) {
	type TestStruct struct {
		Name string
	}

	ts := &TestStruct{Name: "John"}

	// Create a PointerFieldRules with a non-pointer value
	pfr := &PointerFieldRules{
		fieldPtr: ts.Name, // Not a pointer
		rules:    []Rule{Required},
	}

	structValue := reflect.ValueOf(ts).Elem()
	_, _, err := pfr.FindStructField(structValue, 0)

	assert.Error(t, err)
	_, ok := err.(InternalError)
	assert.True(t, ok)

	var errFieldPtr ErrFieldPointer
	assert.True(t, errors.As(err.(InternalError).InternalError(), &errFieldPtr))
}

func TestFieldStruct(t *testing.T) {
	type Address struct {
		Street string
		City   string
	}
	type Person struct {
		Name    string
		Address Address
	}

	p := &Person{
		Name:    "John",
		Address: Address{Street: "Main St", City: "NYC"},
	}

	fr := FieldStruct(&p.Address,
		Field(&p.Address.Street, Required),
		Field(&p.Address.City, Required),
	)

	assert.Len(t, fr.Rules(), 1)
	assert.True(t, fr.validatePtrValue)

	structValue := reflect.ValueOf(p).Elem()
	ft, value, err := fr.FindStructField(structValue, 0)

	assert.NoError(t, err)
	assert.NotNil(t, ft)
	assert.Equal(t, "Address", ft.Name)

	// The value should be a pointer to the Address struct
	addrPtr, ok := value.(*Address)
	assert.True(t, ok)
	assert.Equal(t, "Main St", addrPtr.Street)
	assert.Equal(t, "NYC", addrPtr.City)
}

func TestFindStructField_Detailed(t *testing.T) {
	type Embedded struct {
		EmbeddedField string
	}

	type TestStruct struct {
		Field1 int
		Field2 *int
		Field3 []int
		Field4 [4]int
		field5 int
		Embedded
		Nested struct {
			NestedField string
		}
	}

	var ts TestStruct
	ts.EmbeddedField = "embedded"
	structValue := reflect.ValueOf(&ts).Elem()

	tests := []struct {
		name      string
		fieldPtr  interface{}
		wantFound bool
		wantName  string
	}{
		{
			name:      "find regular field",
			fieldPtr:  &ts.Field1,
			wantFound: true,
			wantName:  "Field1",
		},
		{
			name:      "find pointer field",
			fieldPtr:  &ts.Field2,
			wantFound: true,
			wantName:  "Field2",
		},
		{
			name:      "find slice field",
			fieldPtr:  &ts.Field3,
			wantFound: true,
			wantName:  "Field3",
		},
		{
			name:      "find array field",
			fieldPtr:  &ts.Field4,
			wantFound: true,
			wantName:  "Field4",
		},
		{
			name:      "find unexported field",
			fieldPtr:  &ts.field5,
			wantFound: true,
			wantName:  "field5",
		},
		{
			name:      "find embedded struct field",
			fieldPtr:  &ts.EmbeddedField,
			wantFound: true,
			wantName:  "EmbeddedField",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldValue := reflect.ValueOf(tt.fieldPtr)

			// Skip non-pointer test as it would panic
			if fieldValue.Kind() != reflect.Ptr {
				return
			}

			result := findStructField(structValue, fieldValue)

			if tt.wantFound {
				assert.NotNil(t, result)
				if result != nil {
					assert.Equal(t, tt.wantName, result.Name)
				}
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestFindStructField_WithPointerEmbedded(t *testing.T) {
	type Embedded struct {
		EmbeddedField string
	}

	type TestStruct struct {
		*Embedded
		OtherField string
	}

	ts := TestStruct{
		Embedded:   &Embedded{EmbeddedField: "test"},
		OtherField: "other",
	}

	structValue := reflect.ValueOf(&ts).Elem()

	// Find field in pointer-embedded struct
	result := findStructField(structValue, reflect.ValueOf(&ts.EmbeddedField))
	assert.NotNil(t, result)
	assert.Equal(t, "EmbeddedField", result.Name)
}

func TestNamedFieldRules_Integration(t *testing.T) {
	type TestStruct struct {
		Name  string
		Email string
		Age   int
	}

	tests := []struct {
		name    string
		setup   func() (*TestStruct, []*NamedFieldRules)
		wantErr string
	}{
		{
			name: "valid named fields",
			setup: func() (*TestStruct, []*NamedFieldRules) {
				ts := &TestStruct{Name: "John Doe", Email: "john@example.com", Age: 25}
				return ts, []*NamedFieldRules{
					NamedField("Name", Required),
					NamedField("Email", Required),
					NamedField("Age", Required),
				}
			},
			wantErr: "",
		},
		{
			name: "missing required named field",
			setup: func() (*TestStruct, []*NamedFieldRules) {
				ts := &TestStruct{Name: "", Email: "john@example.com", Age: 25}
				return ts, []*NamedFieldRules{
					NamedField("Name", Required),
				}
			},
			wantErr: "Name: cannot be blank.",
		},
		{
			name: "skip if not found - no error",
			setup: func() (*TestStruct, []*NamedFieldRules) {
				ts := &TestStruct{Name: "John"}
				fr := NamedField("NonExistent", Required)
				fr.SetSkipIfNotFound(true)
				return ts, []*NamedFieldRules{fr}
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, rules := tt.setup()

			// Convert to FieldRules interface
			fieldRules := make([]FieldRules, len(rules))
			for i, r := range rules {
				fieldRules[i] = r
			}

			err := ValidateStructWithContext(context.Background(), ts, fieldRules...)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestPointerFieldRules_Integration(t *testing.T) {
	type Address struct {
		Street string
		City   string
	}
	type Person struct {
		Name    string
		Address Address
	}

	tests := []struct {
		name    string
		setup   func() (*Person, []FieldRules)
		wantErr string
	}{
		{
			name: "valid fields",
			setup: func() (*Person, []FieldRules) {
				p := &Person{
					Name:    "John",
					Address: Address{Street: "Main St", City: "NYC"},
				}
				return p, []FieldRules{
					Field(&p.Name, Required),
					FieldStruct(&p.Address,
						Field(&p.Address.Street, Required),
						Field(&p.Address.City, Required),
					),
				}
			},
			wantErr: "",
		},
		{
			name: "invalid nested struct",
			setup: func() (*Person, []FieldRules) {
				p := &Person{
					Name:    "John",
					Address: Address{Street: "", City: "NYC"},
				}
				return p, []FieldRules{
					FieldStruct(&p.Address,
						Field(&p.Address.Street, Required),
						Field(&p.Address.City, Required),
					),
				}
			},
			wantErr: "Address: (Street: cannot be blank.).",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, rules := tt.setup()
			err := ValidateStructWithContext(context.Background(), p, rules...)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestNamedFieldRules_ValidatePtrValue(t *testing.T) {
	type Inner struct {
		Value string
	}
	type Outer struct {
		Inner Inner
	}

	// Test with validatePtrValue = false (default for NamedField)
	fr1 := NamedField("Inner", Required)
	assert.False(t, fr1.validatePtrValue)

	// Test with validatePtrValue = true (NamedStructField)
	fr2 := NamedStructField("Inner", NamedField("Value", Required))
	assert.True(t, fr2.validatePtrValue)

	outer := &Outer{Inner: Inner{Value: "test"}}
	structValue := reflect.ValueOf(outer).Elem()

	// Test that NamedField returns the value itself (not pointer)
	_, value1, err := fr1.FindStructField(structValue, 0)
	assert.NoError(t, err)
	_, ok := value1.(Inner)
	assert.True(t, ok, "NamedField should return value, not pointer")

	// Test that NamedStructField returns a pointer to the value
	_, value2, err := fr2.FindStructField(structValue, 0)
	assert.NoError(t, err)
	_, ok = value2.(*Inner)
	assert.True(t, ok, "NamedStructField should return pointer to value")
}

func TestPointerFieldRules_ValidatePtrValue(t *testing.T) {
	type Inner struct {
		Value string
	}
	type Outer struct {
		Inner Inner
	}

	// Test with validatePtrValue = false (default for Field)
	outer := &Outer{Inner: Inner{Value: "test"}}
	fr1 := Field(&outer.Inner, Required).(*PointerFieldRules)
	assert.False(t, fr1.validatePtrValue)

	// Test with validatePtrValue = true (FieldStruct)
	fr2 := FieldStruct(&outer.Inner, Field(&outer.Inner.Value, Required))
	assert.True(t, fr2.validatePtrValue)

	structValue := reflect.ValueOf(outer).Elem()

	// Test that Field returns the value itself (not pointer)
	_, value1, err := fr1.FindStructField(structValue, 0)
	assert.NoError(t, err)
	_, ok := value1.(Inner)
	assert.True(t, ok, "Field should return value, not pointer")

	// Test that FieldStruct returns a pointer to the value
	_, value2, err := fr2.FindStructField(structValue, 0)
	assert.NoError(t, err)
	_, ok = value2.(*Inner)
	assert.True(t, ok, "FieldStruct should return pointer to value")
}
