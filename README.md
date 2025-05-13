# Eolas

Eolas is a command line utility for analyzing Kubernetes cluster configurations. It ingests JSON files containing Kubernetes resources and provides analysis capabilities.

## Installation

```
go install github.com/raesene/eolas@latest
```

## Usage

```
eolas [command]
```

### Available Commands

#### Ingest Kubernetes Configuration

```
eolas ingest -f <path-to-json-file> -n <cluster-name>
```

Options:
- `-f, --file` - Path to the JSON file containing Kubernetes cluster configuration (required)
- `-n, --name` - Name to identify the cluster configuration (defaults to timestamp)
- `-s, --storage-dir` - Directory to store parsed configurations (defaults to .eolas)

Example:
```
eolas ingest -f sample_data/sample-kind.json -n kind-cluster
```

#### List Stored Configurations

```
eolas list
```

Options:
- `-s, --storage-dir` - Directory where configurations are stored (defaults to .eolas)

## Development

```
# Clone the repository
git clone https://github.com/raesene/eolas.git
cd eolas

# Build
go build
# or use make
make build

# Run
./eolas
# or use make
make run
```

### Using Make

The project includes a Makefile with common operations:

```
make build     # Build the application
make clean     # Remove build artifacts
make test      # Run tests
make fmt       # Format code
make vet       # Run go vet
make install   # Install the application
make check     # Run all quality checks
make run       # Build and run the application
make help      # Show help message
```

## Project Structure

- `cmd/` - Command implementations
- `pkg/kubernetes/` - Kubernetes configuration parsing
- `pkg/storage/` - Configuration storage

## Features

- Parse Kubernetes cluster configuration from JSON files
- Display resource counts by kind
- Store configurations for later analysis
- List stored configurations