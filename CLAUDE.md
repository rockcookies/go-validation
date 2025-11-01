# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go validation library (`go-validation`) - an actively maintained fork of `ozzo-validation` with full context support and modern features. It provides rule-based validation using Go code (not struct tags) for structs, strings, slices, maps, and arrays.

## Development Commands

### Testing
```bash
# Run all tests with coverage (matches CI)
go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Run single test function
go test -run TestSpecificFunction

# Run tests in specific package
go test ./is/...

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

### Linting
```bash
# Run golangci-lint (CI uses latest v2.x)
golangci-lint run

# Install golangci-lint if not present
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Module Management
```bash
go mod tidy    # Clean dependencies
go mod verify  # Verify dependencies
go mod download # Download dependencies
```

### CI/CD
- **Test workflow**: Race detection + coverage reporting to Codecov
- **Lint workflow**: golangci-lint code quality (runs on main branch and PRs)
- **Release workflow**: Automated releases (runs on version tags)
- **Go version**: 1.20+ (specified in go.mod)

## Code Architecture

### Core Interfaces
- `Rule`: Main validation interface with `Validate(ctx context.Context, value interface{}) error`
- `Validatable`: Interface for self-validation types with `Validate(ctx context.Context) error`
- `RuleFunc`: Function wrapper for creating inline validation rules (`func(ctx context.Context, value interface{}) error`)

### Key Components

**Main Entry Points**
- `validation.go`: `ValidateWithContext()` - primary validation method, `By()` helper for function-to-rule conversion
- `struct.go`: `ValidateStructWithContext()` - struct field validation with field error aggregation
- `field.go`: Field validation system with pointer-based and name-based field access
- `option.go`: Context options system for customizing validation behavior

**Field Validation System** (`field.go`)
- `Field(fieldPtr, rules...)`: Pointer-based field validation (traditional approach)
- `NamedField(name, rules...)`: Name-based field validation for dynamic scenarios
- `FieldStruct(structPtr, fields...)`: Pointer-based nested struct validation
- `NamedStructField(name, fields...)`: Name-based nested struct validation
- `FieldRules` interface: Abstraction for different field validation approaches
- Automatic field discovery using pointer comparison or name matching

**Error System** (`error.go`)
- `Error`: Structured validation errors with code/message/params
- `Errors`: Map-based error collection for field-level validation (implements `json.Marshaler`)
- `InternalError`: Wraps non-validation errors to distinguish from validation failures

**Built-in Rules**: Individual files per rule type (required.go, length.go, minmax.go, match.go, etc.)

### Validation Flow
`ValidateWithContext()` processes in order:
1. Apply each rule sequentially, return on first error
2. If value implements `Validatable`, call its `Validate(ctx)`
3. For collections of validatable elements, validate each element with index-based error keys
4. Handle pointer/interface dereference automatically
5. `Skip` rule terminates validation immediately

### Context System
All validation is context-aware:
- All functions accept `context.Context` (defaults to `Background()` if nil)
- Context passed through all rules and validatable types
- Customizable behavior via `WithOptions()`:
  - `WithValuerFunc()`: Custom value extraction (e.g., sql.Valuer with error handling)
  - `WithGetErrorFieldNameFunc()`: Custom error field name resolution (defaults to `json` tags)
  - `WithFindStructFieldByNameFunc()`: Custom struct field lookup by name

### Development Patterns

**File Organization**:
- Core validation logic in main files (validation.go, struct.go, field.go, error.go, option.go)
- Individual validation rules in separate files (`rulename.go`)
- Comprehensive test coverage in corresponding `*_test.go` files
- String validation rules in `is/` subpackage

**Adding New Rules**:
- Implement `Rule` interface with context support
- Add comprehensive table-driven tests following existing patterns
- Use structured errors with `validation.NewError(code, message)`
- Follow naming convention: `rulename.go`/`rulename_test.go`

**Field Validation Patterns**:
- Prefer pointer-based `Field()` for compile-time safety
- Use `NamedField()` for dynamic field scenarios or when avoiding pointer gymnastics
- Use `FieldStruct()`/`NamedStructField()` for nested struct validation
- All field approaches support the same rule sets and error handling

**Error Handling**:
- All rules return structured `Error` interface types
- Proper nil/pointer handling throughout
- Field errors aggregated in `Errors` map with proper field naming
- Internal errors wrapped separately from validation errors

**Testing**: Table-driven tests for edge cases, each rule has corresponding test file with comprehensive coverage