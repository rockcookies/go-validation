# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go validation library (`go-validation`) - a fork of `ozzo-validation` with full context support and modern features. It provides rule-based validation using Go code (not struct tags) for structs, strings, slices, maps, and arrays.

## Development Commands

### Testing
```bash
# Run all tests with coverage (matches CI)
go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Run single test function
go test -run TestSpecificFunction

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

### Linting
```bash
# Run golangci-lint (CI uses v1.58)
golangci-lint run
```

### Module Management
```bash
go mod tidy    # Clean dependencies
go mod verify  # Verify dependencies
```

### CI/CD
- **Test workflow**: Race detection + coverage reporting
- **Lint workflow**: golangci-lint code quality
- **Release workflow**: Automated releases
- **Go version**: 1.20+ (specified in go.mod)

## Code Architecture

### Core Interfaces
- `Rule`: Main validation interface with `Validate(ctx context.Context, value interface{}) error`
- `Validatable`: Interface for self-validation types with `Validate(ctx context.Context) error`
- `RuleFunc`: Function wrapper for creating inline validation rules

### Key Components

**Main Entry Points**
- `validation.go`: `ValidateWithContext()` - primary validation method, `By()` helper
- `struct.go`: `ValidateStructWithContext()` - struct field validation, `Field()`, `FieldStruct()`
- `option.go`: Context options system for customizing validation behavior

**Error System** (`error.go`)
- `Error`: Structured validation errors with code/message/params
- `Errors`: Map-based error collection for field-level validation
- `InternalError`: Wraps non-validation errors

**Built-in Rules**: Individual files per rule type (required.go, length.go, minmax.go, match.go, etc.)

### Validation Flow
`ValidateWithContext()` processes in order:
1. Apply each rule sequentially, return on first error
2. If value implements `Validatable`, call its `Validate(ctx)`
3. For collections of validatable elements, validate each element
4. Handle pointer/interface dereference automatically
5. `Skip` rule terminates validation

### Context System
All validation is context-aware:
- All functions accept `context.Context` (defaults to `Background()` if nil)
- Context passed through all rules and validatable types
- Customizable behavior via `WithOptions()`:
  - `WithValuerFunc()`: Custom value extraction (e.g., sql.Valuer)
  - `WithGetErrorFieldNameFunc()`: Custom error field name resolution
  - `WithFindStructFieldByNameFunc()`: Custom struct field lookup

### Development Patterns

**File Organization**: Core logic in main files, individual rules in separate files, tests in `*_test.go` files

**Adding New Rules**: Implement `Rule` interface, add comprehensive tests, follow existing naming patterns (`rulename.go`/`rulename_test.go`)

**Error Handling**: All rules return structured `Error` interface types, proper nil/pointer handling throughout

**Testing**: Table-driven tests for edge cases, each rule has corresponding test file