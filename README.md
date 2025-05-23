# Eolas

Eolas is a comprehensive command-line utility for analyzing Kubernetes cluster configurations with advanced storage, comparison, and reporting capabilities. It ingests JSON files containing Kubernetes resources and provides powerful analysis, versioning, and temporal tracking features.

## ✨ Key Features

- **📊 Comprehensive Analysis**: Resource counts, security analysis, and configuration insights
- **🗄️ Dual Storage Backends**: File-based and SQLite storage with versioning
- **📈 Configuration Evolution**: Timeline reports and trend analysis
- **⚖️ Configuration Comparison**: Compare configurations across time and environments
- **🔒 Security Focus**: Privileged containers, capabilities, host access detection
- **📱 Interactive Reports**: Responsive HTML reports with embedded CSS
- **📤 Data Export**: JSON and CSV export for external processing
- **🔄 Data Migration**: Seamless migration between storage backends
- **🧹 Maintenance Tools**: Cleanup and optimization utilities

## 🚀 Quick Start

### Installation

#### Download Pre-built Binaries

Pre-built binaries for Linux, macOS, and Windows are available on the [GitHub Releases page](https://github.com/raesene/eolas/releases).

1. Navigate to the [Releases page](https://github.com/raesene/eolas/releases)
2. Download the appropriate binary for your platform:
   - `eolas_Linux_x86_64.tar.gz` for Linux (64-bit)
   - `eolas_Darwin_x86_64.tar.gz` for macOS (Intel)
   - `eolas_Darwin_arm64.tar.gz` for macOS (Apple Silicon)
   - `eolas_Windows_x86_64.zip` for Windows (64-bit)
3. Extract the binary and place it in your PATH

#### Install from Source

```bash
go install github.com/raesene/eolas@latest
```

### Generate Kubernetes Configuration

Extract your cluster configuration using kubectl:

```bash
kubectl get $(kubectl api-resources --verbs=list -o name | grep -v -e "secrets" -e "componentstatuses" -e "priorityclass" -e "events" | paste -sd, -) --ignore-not-found --all-namespaces -o json > cluster-config.json
```

### Basic Usage

```bash
# Ingest a configuration
eolas ingest -f cluster-config.json -n prod-cluster

# Analyze the configuration
eolas analyze -n prod-cluster --security

# Generate HTML report
eolas analyze -n prod-cluster --security --html -o report.html
```

## 📋 Commands Overview

| Command | Description |
|---------|-------------|
| `ingest` | Ingest Kubernetes configuration JSON files |
| `analyze` | Analyze stored configurations with security insights |
| `list` | List stored configurations and view history |
| `compare` | Compare two configurations to identify differences |
| `timeline` | Generate timeline reports showing configuration evolution |
| `export` | Export analysis data in JSON or CSV format |
| `migrate` | Migrate data between storage backends |
| `cleanup` | Clean up old configurations and optimize storage |
| `version` | Show version and build information |

## 🗄️ Storage Backends

Eolas supports two storage backends:

### File Backend (Default)
- Simple file-based storage
- One configuration per file
- Suitable for basic analysis needs

### SQLite Backend (Advanced)
- Versioned configuration storage
- Configuration history tracking
- Timeline and comparison features
- Pre-computed security analysis

```bash
# Use SQLite backend for advanced features
eolas ingest -f config.json -n prod-cluster --backend sqlite
eolas analyze -n prod-cluster --backend sqlite --security
```

## 📊 Analysis Features

### Resource Analysis
- Resource type counts and distributions
- Cluster inventory and resource utilization
- Namespace-based resource breakdown

### Security Analysis
Comprehensive security analysis including:

#### 🔴 Privileged Containers
Identifies containers running with privileged security context:
```bash
eolas analyze -n cluster --privileged
```

#### 🟡 Linux Capabilities
Detects containers with added Linux capabilities:
```bash
eolas analyze -n cluster --capabilities
```

#### 🟠 Host Namespace Usage
Finds workloads using host namespaces (hostPID, hostIPC, hostNetwork):
```bash
eolas analyze -n cluster --host-namespaces
```

#### 🔶 Host Path Volumes
Identifies workloads mounting host filesystem paths:
```bash
eolas analyze -n cluster --host-path
```

#### 🔒 Combined Security Analysis
Run all security checks at once:
```bash
eolas analyze -n cluster --security
```

## 📈 Configuration Evolution & Comparison

### Configuration History
View configuration versions over time (SQLite backend):
```bash
eolas list --backend sqlite --history --name prod-cluster
```

### Configuration Comparison
Compare two configurations to identify changes:
```bash
eolas compare --backend sqlite --config1 uuid1 --config2 uuid2
eolas compare --backend sqlite --config1 uuid1 --config2 uuid2 --html -o comparison.html
```

### Timeline Reports
Generate interactive timeline reports showing configuration evolution:
```bash
eolas timeline --name prod-cluster
eolas timeline --name prod-cluster -o timeline-report.html
```

Timeline reports include:
- Configuration version timeline with change tracking
- Resource trend analysis (increasing/decreasing/stable)
- Security posture evolution
- Current vs previous snapshot comparison

## 📤 Data Export

Export analysis data for external processing:

### JSON Export
```bash
# Export complete analysis
eolas export --name cluster --format json

# Export only security findings
eolas export --name cluster --format json --type security

# Export to custom file
eolas export --name cluster --format json -o analysis.json
```

### CSV Export
```bash
# Export resource counts
eolas export --name cluster --format csv --type resources

# Export security findings
eolas export --name cluster --format csv --type security
```

## 🔄 Data Migration

Migrate configurations between storage backends:

```bash
# Preview migration (dry run)
eolas migrate --from file --to sqlite --dry-run

# Migrate from file to SQLite for advanced features
eolas migrate --from file --to sqlite

# Force overwrite existing configurations
eolas migrate --from file --to sqlite --force
```

## 🧹 Maintenance & Cleanup

### Clean Up Old Configurations
```bash
# Remove configurations older than 30 days
eolas cleanup --older-than 30d --dry-run

# Keep only latest 5 versions (SQLite)
eolas cleanup --backend sqlite --keep-versions 5

# Clean specific configuration
eolas cleanup --name old-cluster --older-than 7d
```

### Storage Optimization
```bash
# Dry run to see what would be cleaned
eolas cleanup --backend sqlite --keep-versions 3 --dry-run

# Actual cleanup
eolas cleanup --backend sqlite --keep-versions 3
```

## 📱 HTML Reports

Eolas generates professional, responsive HTML reports with:

- **📊 Overview Dashboard**: Resource counts and security summary
- **🔍 Tabbed Interface**: Easy navigation between analysis types
- **🎨 Color-coded Results**: Visual indicators for security issues
- **📱 Responsive Design**: Works on desktop, tablet, and mobile
- **📦 Self-contained**: No external dependencies

### Report Types

#### Analysis Reports
```bash
eolas analyze -n cluster --security --html -o security-report.html
```

#### Comparison Reports
```bash
eolas compare --config1 id1 --config2 id2 --html -o comparison.html
```

#### Timeline Reports
```bash
eolas timeline --name cluster -o evolution-report.html
```

## 🔧 Advanced Configuration

### Storage Directory Configuration
```bash
# Use custom storage directory
eolas ingest -f config.json -n cluster -s /path/to/storage

# Use current directory instead of home
eolas ingest -f config.json -n cluster --use-home=false
```

### Output Customization
```bash
# Output to stdout
eolas export --name cluster --format json -o -

# Custom filename with automatic extension
eolas timeline --name cluster -o my-timeline  # Creates my-timeline.html
```

## 🛠️ Development

### Build from Source
```bash
git clone https://github.com/raesene/eolas.git
cd eolas
go build
```

### Using Make
```bash
make build     # Build the application
make test      # Run tests
make fmt       # Format code
make vet       # Run go vet
make clean     # Remove build artifacts
make help      # Show help message
```

### Project Structure
```
eolas/
├── cmd/           # Command implementations (analyze, ingest, compare, etc.)
├── pkg/
│   ├── kubernetes/    # Kubernetes configuration parsing and analysis
│   ├── storage/       # Storage backends (file, SQLite)
│   └── output/        # Output formatters (HTML, timeline)
├── sample_data/   # Sample Kubernetes configurations
└── docs/          # Documentation website
```

## 📊 Version Information

Get detailed build and dependency information:

```bash
# Basic version
eolas version

# Detailed version with dependencies and features
eolas version --detailed
```

## 🤝 Contributing

We welcome contributions! Please see our contributing guidelines and feel free to:

- Report bugs or request features via GitHub Issues
- Submit pull requests for improvements
- Share feedback and suggestions

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🔗 Links

- **GitHub Repository**: [https://github.com/raesene/eolas](https://github.com/raesene/eolas)
- **Documentation**: [GitHub Pages Documentation](https://raesene.github.io/eolas)
- **Releases**: [GitHub Releases](https://github.com/raesene/eolas/releases)

---

*Eolas (pronounced "oh-las") is an Irish word meaning "knowledge" or "information" - fitting for a tool designed to provide insights into your Kubernetes configurations.*