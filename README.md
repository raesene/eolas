# Eolas

Eolas is a command line utility for analyzing Kubernetes cluster configurations. It ingests JSON files containing Kubernetes resources and provides analysis capabilities.

## Installation

### Download Pre-built Binaries

Pre-built binaries for Linux, macOS, and Windows are available on the [GitHub Releases page](https://github.com/raesene/eolas/releases).

1. Navigate to the [Releases page](https://github.com/raesene/eolas/releases)
2. Download the appropriate binary for your platform:
   - `eolas_Linux_x86_64.tar.gz` for Linux (64-bit)
   - `eolas_Darwin_x86_64.tar.gz` for macOS (Intel)
   - `eolas_Darwin_arm64.tar.gz` for macOS (Apple Silicon)
   - `eolas_Windows_x86_64.zip` for Windows (64-bit)
3. Extract the binary and place it in your PATH

### Install from Source

If you prefer to install from source:

```
go install github.com/raesene/eolas@latest
```

## Generating Kubernetes Configuration JSON

To generate the JSON configuration file for ingestion into Eolas, use the following `kubectl` command:

```bash
kubectl get $(kubectl api-resources --verbs=list -o name | grep -v -e "secrets" -e "componentstatuses" -e "priorityclass" -e "events" | paste -sd, -) --ignore-not-found --all-namespaces -o json > cluster-config.json
```

This command:

1. Gets a list of all available API resources that support the `list` verb
2. Excludes sensitive or high-volume resources (secrets, componentstatuses, priorityclass, events)
3. Retrieves all instances of these resources across all namespaces
4. Outputs the result as a single JSON file

After generating the JSON file, you can ingest it into Eolas:

```bash
eolas ingest -f cluster-config.json -n my-cluster
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
- `-s, --storage-dir` - Directory to store parsed configurations (defaults to .eolas in home directory)
- `--use-home` - Whether to use the home directory for storage (defaults to true)

Example:
```
eolas ingest -f sample_data/sample-kind.json -n kind-cluster
```

#### List Stored Configurations

```
eolas list
```

Options:
- `-s, --storage-dir` - Directory where configurations are stored (defaults to .eolas in home directory)
- `--use-home` - Whether to use the home directory for storage (defaults to true)

#### Analyze a Cluster Configuration

```
eolas analyze -n <cluster-name> [flags]
```

Options:
- `-n, --name` - Name of the cluster configuration to analyze (required)
- `-s, --storage-dir` - Directory where configurations are stored (defaults to .eolas in home directory)
- `--use-home` - Whether to use the home directory for storage (defaults to true)
- `--security` - Run security-focused analysis on the cluster configuration
- `--privileged` - Check for privileged containers in the cluster configuration
- `--capabilities` - Check for containers with added Linux capabilities
- `--host-namespaces` - Check for workloads using host namespaces
- `--host-path` - Check for workloads using hostPath volumes
- `--html` - Generate HTML output
- `-o, --output` - File to write output to (default is stdout)

Examples:
```bash
# Basic resource count analysis
eolas analyze -n kind-cluster

# Full security analysis (includes all security checks)
eolas analyze -n kind-cluster --security

# Specific security analysis for privileged containers
eolas analyze -n kind-cluster --privileged

# Specific security analysis for containers with added capabilities
eolas analyze -n kind-cluster --capabilities

# Specific security analysis for workloads using host namespaces
eolas analyze -n kind-cluster --host-namespaces

# Specific security analysis for workloads using hostPath volumes
eolas analyze -n kind-cluster --host-path

# Generate an HTML report and save to file
eolas analyze -n kind-cluster --html -o cluster-report.html

# Generate a comprehensive HTML security report
eolas analyze -n kind-cluster --security --html -o security-report.html
```

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
- Analyze cluster configurations for detailed insights
  - Resource type counts and distributions
  - Security analysis capabilities:
    - Identification of privileged containers
    - Detection of containers with added Linux capabilities
    - Discovery of workloads using host namespaces (hostPID, hostIPC, hostNetwork)
    - Identification of containers exposing host ports
    - Detection of workloads using hostPath volumes

## Security Analysis

Eolas provides several security analysis features to help identify potential security risks in your Kubernetes cluster configurations. These features can be used individually or together using the `--security` flag.

### Privileged Containers

Privileged containers have full access to the host's kernel capabilities and device nodes, similar to root access on the host. These pose significant security risks and should be carefully reviewed.

```bash
eolas analyze -n my-cluster --privileged
```

### Linux Capabilities

Linux capabilities provide fine-grained control over privileged operations. Containers with added capabilities have elevated privileges that may pose security risks. Particularly dangerous capabilities include: CAP_SYS_ADMIN, CAP_NET_ADMIN, CAP_SYS_PTRACE, and CAP_NET_RAW.

```bash
eolas analyze -n my-cluster --capabilities
```

### Host Namespaces

Host namespaces provide containers with access to the host's resources, reducing isolation between containers and the host system. Each namespace type has specific security implications:

- **hostPID**: Allows visibility of all processes on the host system
- **hostIPC**: Enables shared memory access with the host and all containers
- **hostNetwork**: Provides direct access to the host's network interfaces
- **hostPorts**: Exposes ports directly on the host's network interfaces

```bash
eolas analyze -n my-cluster --host-namespaces
```

### Host Path Volumes

hostPath volumes allow pods to mount files or directories from the host node's filesystem directly into the pod. This poses significant security risks as it enables containers to access and potentially modify sensitive areas of the host filesystem. Risks include:

- Read access to sensitive host files
- Potential modification of host system files (when not mounted read-only)
- Persistence across pod restarts, potentially allowing data exfiltration
- Potential for privilege escalation through the host filesystem

```bash
eolas analyze -n my-cluster --host-path
```

### Combined Security Analysis

Run all security checks at once using the `--security` flag:

```bash
eolas analyze -n my-cluster --security
```

## HTML Reports

Eolas can generate interactive HTML reports with all analysis results. These reports include:

- Overview dashboard with resource counts and security findings
- Tabbed interface for easy navigation between different analysis types
- Color-coded tables for better visualization of security issues
- Responsive design that works on all devices
- Embedded CSS and JavaScript (no external dependencies)

Generate HTML reports using the `--html` flag:

```bash
# Generate a basic HTML report
eolas analyze -n my-cluster --html -o report.html

# Generate a comprehensive security report
eolas analyze -n my-cluster --security --html -o security-report.html
```

The HTML reports are self-contained single files that can be easily shared and viewed in any web browser.

## Releases

The project uses GitHub Actions with GoReleaser to automatically build and publish releases. When a new tag is pushed, GoReleaser will build binaries for multiple platforms and publish them as a GitHub release.

To create a new release:

```bash
# Update code and commit changes
git commit -am "Your changes"

# Tag a new version
git tag -a v0.1.0 -m "First release"

# Push tag to GitHub
git push origin v0.1.0
```

The GitHub Action will automatically build and publish the release.