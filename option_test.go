package validation

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	assert.NotNil(t, opts)
	assert.NotNil(t, opts.ValuerFunc())
	assert.NotNil(t, opts.GetErrorFieldNameFunc())
}

func TestWithValuerFunc(t *testing.T) {
	customValuerCalled := false
	customValuer := func(v any) (any, bool) {
		customValuerCalled = true
		// Return false to avoid infinite recursion
		return v, false
	}

	ctx := WithOptions(context.Background(), WithValuerFunc(customValuer))
	opts := GetOptions(ctx)

	assert.NotNil(t, opts.ValuerFunc())

	// Test if custom ValuerFunc is called
	value, _ := indirectWithOptions("test", opts)
	assert.True(t, customValuerCalled)
	assert.Equal(t, "test", value)
}

func TestWithGetErrorFieldNameFunc(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	// Custom field name function: use field name instead of JSON tag
	customFunc := func(f *reflect.StructField) string {
		return "field_" + f.Name
	}

	s := TestStruct{Name: "", Email: ""}
	ctx := WithOptions(context.Background(), WithGetErrorFieldNameFunc(customFunc))

	err := ValidateStructWithContext(ctx, &s,
		Field(&s.Name, Required),
		Field(&s.Email, Required),
	)

	if assert.NotNil(t, err) {
		errs := err.(Errors)
		_, hasNameField := errs["field_Name"]
		_, hasEmailField := errs["field_Email"]
		assert.True(t, hasNameField, "Expected field_Name in errors")
		assert.True(t, hasEmailField, "Expected field_Email in errors")
	}
}

func TestWithOptions(t *testing.T) {
	// Test with nil context
	ctx1 := WithOptions(nil, WithValuerFunc(DefaultValuer))
	assert.NotNil(t, ctx1)
	opts1 := GetOptions(ctx1)
	assert.NotNil(t, opts1)

	// Test with multiple Options
	customValuer := func(v any) (any, bool) { return v, false }
	customFieldName := func(f *reflect.StructField) string { return f.Name }

	ctx2 := WithOptions(context.Background(),
		WithValuerFunc(customValuer),
		WithGetErrorFieldNameFunc(customFieldName),
	)

	opts2 := GetOptions(ctx2)
	assert.NotNil(t, opts2.ValuerFunc())
	assert.NotNil(t, opts2.GetErrorFieldNameFunc())
}

func TestGetOptions(t *testing.T) {
	// Test nil context, should return defaultOptions
	opts1 := GetOptions(nil)
	assert.NotNil(t, opts1)
	assert.Equal(t, defaultOptions, opts1)

	// Test context without options, should return defaultOptions
	ctx := context.Background()
	opts2 := GetOptions(ctx)
	assert.NotNil(t, opts2)
	assert.Equal(t, defaultOptions, opts2)

	// Test context with options
	customCtx := WithOptions(context.Background(), WithValuerFunc(DefaultValuer))
	opts3 := GetOptions(customCtx)
	assert.NotNil(t, opts3)
	assert.NotEqual(t, defaultOptions, opts3)
}

func TestGetOpts(t *testing.T) {
	// Test getOpts function
	opts1 := getOpts(nil)
	assert.Equal(t, defaultOptions, opts1)

	ctx := context.Background()
	opts2 := getOpts(ctx)
	assert.Equal(t, defaultOptions, opts2)

	customCtx := WithOptions(context.Background(), WithValuerFunc(DefaultValuer))
	opts3 := getOpts(customCtx)
	assert.NotNil(t, opts3)
	assert.NotEqual(t, defaultOptions, opts3)
}

func TestOptionsInterface(t *testing.T) {
	// Ensure options implements Options interface
	var _ Options = (*options)(nil)

	opts := &options{
		valuerFunc:            DefaultValuer,
		getErrorFieldNameFunc: DefaultGetErrorFieldName,
	}

	assert.NotNil(t, opts.ValuerFunc())
	assert.NotNil(t, opts.GetErrorFieldNameFunc())
}

func TestCustomValuerWithStruct(t *testing.T) {
	type CustomType struct {
		Value string
	}

	type TestStruct struct {
		Field CustomType
	}

	// Custom Valuer: handle CustomType
	customValuer := func(v any) (any, bool) {
		if ct, ok := v.(CustomType); ok {
			return ct.Value, true
		}
		return v, false
	}

	s := TestStruct{Field: CustomType{Value: ""}}
	ctx := WithOptions(context.Background(), WithValuerFunc(customValuer))

	// Validate using custom valuer
	err := ValidateStructWithContext(ctx, &s,
		Field(&s.Field, Required),
	)

	// Field.Value is empty string, should fail validation
	if assert.NotNil(t, err) {
		errs := err.(Errors)
		assert.Contains(t, errs, "Field")
	}
}

func TestDefaultValuerWithSqlNullTypes(t *testing.T) {
	// Test DefaultValuer handles sql.Null* types
	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
		isNil    bool
	}{
		{
			name:     "sql.NullString valid",
			value:    sql.NullString{String: "test", Valid: true},
			expected: "test",
			isNil:    false,
		},
		{
			name:     "sql.NullString invalid",
			value:    sql.NullString{String: "test", Valid: false},
			expected: nil,
			isNil:    true,
		},
		{
			name:     "sql.NullInt64 valid",
			value:    sql.NullInt64{Int64: 123, Valid: true},
			expected: int64(123),
			isNil:    false,
		},
		{
			name:     "sql.NullInt64 invalid",
			value:    sql.NullInt64{Int64: 123, Valid: false},
			expected: nil,
			isNil:    true,
		},
		{
			name:     "regular string",
			value:    "test",
			expected: "test",
			isNil:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, isNil := Indirect(test.value)
			assert.Equal(t, test.expected, result)
			assert.Equal(t, test.isNil, isNil)
		})
	}
}

func TestOptionsImmutability(t *testing.T) {
	// Test WithOptions does not modify original context's options
	ctx1 := WithOptions(context.Background(), WithValuerFunc(func(v any) (any, bool) {
		return "first", true
	}))

	ctx2 := WithOptions(ctx1, WithValuerFunc(func(v any) (any, bool) {
		return "second", true
	}))

	opts1 := GetOptions(ctx1)
	opts2 := GetOptions(ctx2)

	// Two contexts should have different options
	val1, _ := opts1.ValuerFunc()("test")
	val2, _ := opts2.ValuerFunc()("test")

	assert.Equal(t, "first", val1)
	assert.Equal(t, "second", val2)
}

func TestGetErrorFieldNameFuncIntegration(t *testing.T) {
	type User struct {
		FirstName string `json:"first_name" xml:"firstName"`
		LastName  string `json:"last_name" xml:"lastName"`
	}

	// Test using XML tag as error field name
	customFunc := func(f *reflect.StructField) string {
		if tag := f.Tag.Get("xml"); tag != "" {
			return tag
		}
		return f.Name
	}

	u := User{FirstName: "", LastName: ""}
	ctx := WithOptions(context.Background(), WithGetErrorFieldNameFunc(customFunc))

	err := ValidateStructWithContext(ctx, &u,
		Field(&u.FirstName, Required),
		Field(&u.LastName, Required),
	)

	if assert.NotNil(t, err) {
		errs := err.(Errors)
		_, hasFirstName := errs["firstName"]
		_, hasLastName := errs["lastName"]
		assert.True(t, hasFirstName, "Expected firstName (from XML tag) in errors")
		assert.True(t, hasLastName, "Expected lastName (from XML tag) in errors")
	}
}
