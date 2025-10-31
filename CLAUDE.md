# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go validation library (`go-validation`) that provides configurable and extensible data validation capabilities. It's an actively maintained fork of the original `ozzo-validation` library with added context support and modern features.

### Key Features
- Rule-based validation using Go code (not struct tags)
- Support for various data types: structs, strings, slices, maps, arrays
- Context-aware validation
- Customizable error messages with internationalization support
- Built-in validation rules and extensible rule system

## Development Commands

### Testing
```bash
# Run all tests with coverage (matches CI configuration)
go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Run tests for current package only
go test .

# Run tests for specific package
go test ./package_name

# Run specific test function
go test -run TestSpecificFunction

# Run tests with verbose output
go test -v ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Linting
```bash
# Run golangci-lint (used in CI, version v1.58)
golangci-lint run

# Lint specific file or directory
golangci-lint run validation.go
golangci-lint run ./...

# Run with specific configuration
golangci-lint run -c .golangci.yml
```

### Module Management
```bash
# Download dependencies
go mod download

# Tidy dependencies (remove unused)
go mod tidy

# Verify dependencies
go mod verify

# Update dependencies
go get -u ./...
go mod tidy
```

### Building
```bash
# Build the package (no main file, so typically used for testing)
go build ./...

# Build with specific target
GOOS=linux GOARCH=amd64 go build ./...

# Install the package locally
go install .
```

### CI/CD Configuration
The project uses GitHub Actions for CI/CD:
- **Test workflow** (`.github/workflows/test.yaml`): Runs tests with race detection and coverage reporting
- **Lint workflow** (`.github/workflows/lint.yaml`): Runs golangci-lint for code quality
- **Release workflow** (`.github/workflows/release.yaml`): Handles automated releases

### Go Version
- **Minimum required**: Go 1.20 (as specified in `go.mod`)
- **CI uses**: Version specified in `go.mod` file automatically

## Code Architecture

### Core Interfaces
- `Rule`: Main validation interface with `Validate(ctx context.Context, value interface{}) error`
- `Validatable`: Interface for types that support self-validation with `Validate(ctx context.Context) error`
- `RuleFunc`: Function type for creating inline validation rules

### Key Components

#### Main Validation Functions (`validation.go:38-156`)
- `ValidateWithContext()`: Primary validation method with context support (context-aware throughout)
- `By()`: Helper function to wrap RuleFunc into Rule interface

#### Struct Validation (`struct.go:43-121`)
- `ValidateStructWithContext()`: Validates struct fields with context
- `ValidateStruct()`: Simplified struct validation without context
- `Field()`: Associates validation rules with struct fields
- `FieldStruct()`: Validates nested struct fields

#### Error Handling (`error.go`)
- `Error`: Interface for validation errors with code, message, and params
- `Errors`: Map-based error collection for field-level errors
- `InternalError`: Wraps non-validation errors that should not be treated as validation failures

#### Built-in Rule Files
- `required.go`: Required/NotNil/Empty rules
- `length.go`: Length/RuneLength rules
- `minmax.go`: Min/Max rules for numeric types and dates
- `match.go`: Regular expression matching
- `in.go`/`not_in.go`: Value inclusion/exclusion rules
- `each.go`: Apply rules to slice/map elements
- `when.go`: Conditional validation with When/Else logic
- `date.go`: Date format validation
- `multipleof.go`: Multiple of validation
- `string.go`: String-specific validation rules
- `absent.go`: Empty value validation

### Validation Flow (`validation.go:51-89`)
The `ValidateWithContext()` function follows this validation order:
1. **Rule Application**: Apply each rule in sequence, returning on first error
2. **Validatable Interface**: If value implements `Validatable`, call its `Validate(ctx context.Context)` method
3. **Collection Validation**: For maps/slices/arrays of validatable elements, validate each element and collect errors
4. **Pointer Handling**: Handle pointer dereference and interface types automatically
5. **Skip Rule**: Special `Skip` rule stops all further validation when encountered

### Context Support
The library is fully context-aware throughout:
- All main validation functions accept `context.Context` parameter
- Context is passed to all rules that implement the `Rule` interface
- Context is automatically used when validating `Validatable` types
- If no context is provided, `context.Background()` is used as default

## Testing Strategy

- Each validation rule has corresponding `*_test.go` files
- Tests cover edge cases, error conditions, and normal operation
- Context-aware validation is tested where applicable
- Use table-driven tests for multiple test cases

## Development Notes

### File Organization
- Core validation logic in `validation.go`, `struct.go`, `error.go`
- Individual rule implementations in separate files (e.g., `required.go`, `length.go`)
- Test files mirror source files with `_test.go` suffix
- Utility functions in `util.go`

### Error Message Customization
- All rules support `.Error()` method for custom messages
- Error templates support parameter substitution
- Error codes are immutable and support internationalization

### Adding New Rules
- Implement `Rule` interface with `Validate(ctx context.Context, value interface{}) error` method
- Consider context support by implementing context-aware validation
- Add comprehensive tests including edge cases in corresponding `*_test.go` files
- Follow existing naming and file organization patterns (e.g., `rulename.go` and `rulename_test.go`)
- Use helper functions like `By()` to wrap validation functions as rules

### Key Implementation Patterns
- **Error Handling**: All validation rules return structured errors implementing the `Error` interface
- **Reflection**: The library uses reflection for type checking and collection validation
- **Nil Safety**: Proper handling of nil values, pointers, and interfaces throughout
- **Error Collection**: `Errors` type collects multiple validation errors with field/key mapping