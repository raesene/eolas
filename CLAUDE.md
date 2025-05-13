# Eolas Project Guide for Claude Code

This document provides context and guidance for Claude Code when working with the Eolas project.

## Project Overview

Eolas is a command-line utility for analyzing Kubernetes cluster configurations. It ingests JSON files containing Kubernetes resources, stores them, and provides analysis capabilities.

## Project Structure

- `cmd/` - Command implementations using Cobra
  - `root.go` - Main command and help text
  - `ingest.go` - Functionality to ingest Kubernetes configurations
  - `list.go` - Lists stored configurations
  - `analyze.go` - Analyzes stored configurations
  - `version.go` - Displays version information
- `pkg/` - Packages containing core functionality
  - `kubernetes/` - Kubernetes configuration parsing
    - `types.go` - Kubernetes resource type definitions
    - `parser.go` - Functions for parsing and analyzing configurations
  - `storage/` - Configuration storage
    - `filestore.go` - File-based storage implementation
- `.github/` - GitHub-related files
  - `workflows/` - GitHub Actions workflows
    - `release.yml` - Workflow for automated releases
    - `build.yml` - Workflow for CI builds and tests
  - `PULL_REQUEST_TEMPLATE/` - Templates for pull requests

## Code Organization

### Command Structure Pattern

Each command follows a consistent pattern:

1. **Variable Declaration**:
   ```go
   var (
     commandFlag1 string
     commandFlag2 bool
   )
   ```

2. **Command Definition**:
   ```go
   var commandCmd = &cobra.Command{
     Use:   "command",
     Short: "Short description",
     Long:  `Longer description.`,
     Run: func(cmd *cobra.Command, args []string) {
       // Command logic
     },
   }
   ```

3. **Command Registration**:
   ```go
   func init() {
     rootCmd.AddCommand(commandCmd)
     commandCmd.Flags().StringVarP(&commandFlag1, "flag", "f", "default", "flag description")
   }
   ```

### Storage Directory Determination Pattern

Commands that access stored data use a consistent pattern to determine the storage directory:

```go
// Determine storage directory
var storeDir string
if providedStorageDir != "" {
  // Use explicitly provided storage directory
  storeDir = providedStorageDir
} else if useHomeDir {
  // Use .eolas in home directory
  homeDir, err := os.UserHomeDir()
  if err != nil {
    fmt.Fprintf(os.Stderr, "Error determining home directory: %v\n", err)
    os.Exit(1)
  }
  storeDir = filepath.Join(homeDir, ".eolas")
} else {
  // Use default .eolas in current directory
  storeDir = ".eolas"
}
```

### Analysis Functions

The project uses specific analysis functions in `pkg/kubernetes`:

- `ParseConfig` - Parses JSON data into Kubernetes structures
- `GetResourceCounts` - Counts resources by type

## Common Operations

### Build and Run the Application

```bash
go build
./eolas
# Or use the Makefile
make build
make run
```

### Add a New Command

1. Create a new file in the `cmd/` directory (e.g., `newcmd.go`)
2. Follow the command structure pattern above
3. Implement the command logic
4. Register the command in the `init()` function
5. Update help text in `root.go` to include the new command
6. Update the README.md with command documentation

### Add a New Analysis Feature

1. For resource-specific analysis:
   - Add a new function in `pkg/kubernetes/parser.go`
   - Call this function from the appropriate command

2. For a new analysis command:
   - Create a new command file in `cmd/`
   - Follow the command structure pattern
   - Implement the analysis logic using existing kubernetes package functions or adding new ones

### Storage Considerations

- All persistent storage is done through the `pkg/storage` package
- Configurations are stored in `.eolas` in the user's home directory by default
- Custom storage locations can be specified with the `-s/--storage-dir` flag

## Release Process

The project uses GitHub Actions with GoReleaser to automatically build and publish releases:

1. Update code and commit changes
2. Tag a new version: `git tag -a v0.x.y -m "Release message"`
3. Push the tag: `git push origin v0.x.y`
4. The GitHub Action will automatically build and publish the release

## Important Considerations

- **JSON Handling**: The application handles large JSON files. Be mindful of memory usage.
- **Error Handling**: Use the consistent pattern of printing errors to stderr and exiting with code 1.
- **Command Structure**: Follow the established pattern for new commands.
- **Storage Location**: Default storage is in the user's home directory at `~/.eolas`.
- **Version Information**: Version is injected at build time via ldflags.

## Development Guidelines

- Follow Go best practices (run `go fmt` and `go vet` before committing)
- Use the Makefile for common operations: `make fmt`, `make vet`, `make test`
- Maintain backward compatibility for existing commands
- Add appropriate documentation for new features
- Follow existing error handling patterns

## Testing

When implementing tests:
- Unit tests should cover core functionality
- Use Go's standard testing package
- Place tests in the same package as the code being tested
- Use meaningful test names describing what's being tested

## Adding New Analysis Features

When adding new analysis features:
1. Consider whether the feature belongs in an existing command or needs a new command
2. For resource-specific analysis, add functions to `pkg/kubernetes/parser.go`
3. For UI formatting, follow the existing pattern of aligned columns and clear headers
4. Update help text and examples to demonstrate the new functionality