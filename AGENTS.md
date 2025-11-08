# Agent Instructions for Axon Codebase

This document provides guidelines for AI agents working on this Go project.

## Build, Lint, and Test

- **Build:** `go build`
- **Test All:** `go test ./...`
- **Test Single File:** `go test path/to/file_test.go`
- **Lint/Format:** `go fmt ./...` and `go vet ./...`

## Code Style Guidelines

- **Formatting:** Strictly follow `gofmt` standards.
- **Imports:** Use standard Go import grouping:
  1. Standard library
  2. Third-party packages
  3. Internal project modules
- **Naming:** Adhere to Go's `camelCase` for internal and `PascalCase` for exported identifiers.
- **Error Handling:** Handle all errors explicitly. Use the custom `utils.HandleError(err)` function for top-level error handling in the `cmd` package. In other packages, return errors to the caller.
- **Types:** Use structs for complex data structures. Avoid global variables where possible.
- **Concurrency:** No specific concurrency patterns were observed, but if needed, use channels and goroutines safely.
