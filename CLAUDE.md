# Eolas Project Guide for Claude Code

This document provides context and guidance for Claude Code when working with the Eolas project.

## Project Overview

Eolas is a command-line utility for analyzing Kubernetes cluster configurations. It ingests JSON files containing Kubernetes resources and provides analysis capabilities.

## Project Structure

- `cmd/` - Command implementations using Cobra
  - `root.go` - Main command
  - `ingest.go` - Functionality to ingest Kubernetes configurations
  - `list.go` - Lists stored configurations
- `pkg/` - Packages containing core functionality
  - `kubernetes/` - Kubernetes configuration parsing
    - `types.go` - Kubernetes resource type definitions
    - `parser.go` - Functions for parsing and analyzing configurations
  - `storage/` - Configuration storage
    - `filestore.go` - File-based storage implementation

## Common Operations

### Build and Run the Application

```bash
go build
./eolas
```

### Run Tests (when implemented)

```bash
go test ./...
```

### Add a New Command

1. Create a new file in the `cmd/` directory
2. Implement the command using Cobra
3. Register the command in the `init()` function
4. Update `root.go` to include the new command in help text

### Add New Kubernetes Resource Types

1. Add new type definitions in `pkg/kubernetes/types.go`
2. Update the parsing logic in `pkg/kubernetes/parser.go` if needed

## Important Considerations

- **JSON Handling**: The application handles large JSON files. Be mindful of memory usage.
- **Error Handling**: Maintain consistent error handling and user feedback.
- **Command Structure**: Follow the established pattern for new commands.
- **Storage**: Files are stored in the `.eolas` directory by default.

## Development Guidelines

- Follow Go best practices (run `go fmt` and `go vet` before committing)
- Maintain backward compatibility for existing commands
- Add appropriate documentation for new features
- Follow existing error handling patterns

## Testing

When implementing tests:
- Unit tests should cover core functionality
- Use Go's standard testing package
- Place tests in the same package as the code being tested
- Use meaningful test names describing what's being tested