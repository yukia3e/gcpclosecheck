# gcpclosecheck

[![Go Report Card](https://goreportcard.com/badge/github.com/yukia3e/gcpclosecheck)](https://goreportcard.com/report/github.com/yukia3e/gcpclosecheck)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Go static analysis tool that detects missing resource cleanup (`Close`, `Stop`, `Cancel`) for GCP resources.

[日本語版](README.ja.md) | English

## 🔍 Overview

`gcpclosecheck` is a static analysis tool that automatically detects locations where resources are not properly released in code using the Google Cloud Platform (GCP) Go SDK.

### Detection Targets

- **GCP Clients**: Missing `defer client.Close()`
- **Spanner**: Missing cleanup for Client, Transaction, RowIterator
- **Cloud Storage**: Missing cleanup for Client, Reader, Writer  
- **Pub/Sub**: Missing Client cleanup
- **Vision API**: Missing Client cleanup
- **Firebase Admin SDK**: Missing Database, Firestore client cleanup
- **reCAPTCHA**: Missing Client cleanup
- **Context**: Missing `cancel()` for `context.WithCancel`, `WithTimeout`, `WithDeadline`

## ⚡ Features

- **Fast**: High-speed processing with lightweight AST analysis
- **Accurate**: Minimizes false positives/negatives with escape analysis
- **Comprehensive**: Supports 6 GCP services + Context
- **Extensible**: Add custom rules via YAML configuration
- **go vet Integration**: Integrates into existing workflows with `-vettool` option
- **Auto-fix**: Automatic `defer` statement addition via SuggestedFix

## 🚀 Installation

```bash
go install github.com/yukia3e/gcpclosecheck/cmd/gcpclosecheck@latest
```

## 📖 Usage

### Basic Execution

```bash
# Analyze single file
gcpclosecheck main.go

# Analyze entire package  
gcpclosecheck ./...

# Analyze specific directory
gcpclosecheck ./internal/...
```

### Integration with go vet

```bash
go vet -vettool=$(which gcpclosecheck) ./...
```

### Options

```bash
gcpclosecheck [options] [packages]

Options:
  -V, --version          Show version
  -fix                   Apply automatic fixes  
  -json                  Output in JSON format
  -gcpdebug              Enable debug mode
  -gcpconfig string      Specify configuration file path
```

## 💡 Examples

### ❌ Problematic Code

```go
package main

import (
    "context"
    "cloud.google.com/go/spanner"
)

func badExample(ctx context.Context) error {
    client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
    if err != nil {
        return err
    }
    // ❌ Missing defer client.Close()

    ctx, cancel := context.WithCancel(ctx)  
    // ❌ Missing defer cancel()
    
    return nil
}
```

### ✅ Fixed Code

```go
package main

import (
    "context"
    "cloud.google.com/go/spanner"  
)

func goodExample(ctx context.Context) error {
    client, err := spanner.NewClient(ctx, "projects/test/instances/test/databases/test")
    if err != nil {
        return err
    }
    defer client.Close() // ✅ Correct

    ctx, cancel := context.WithCancel(ctx)
    defer cancel() // ✅ Correct
    
    return nil  
}
```

### 🔧 Execution Result

```bash
$ gcpclosecheck ./examples/bad.go
./examples/bad.go:12:2: GCP resource client 'client' missing cleanup method (Close)
./examples/bad.go:15:17: Context cancel function should be called with defer
```

## ⚙️ Configuration

### Custom Configuration File

```yaml
# .gcpclosecheck.yaml
services:
  myservice:
    packages:
      - "github.com/myorg/myservice"
    resource_types:
      MyClient:
        creation_functions: ["NewMyClient"]
        cleanup_method: "Close"
        cleanup_required: true
```

## 🏗️ Development & Build

### Prerequisites

- Go 1.21+
- Git

### Build

```bash
git clone https://github.com/yukia3e/gcpclosecheck.git
cd gcpclosecheck
make build
```

### Running Tests

```bash
# Run all tests
make test

# E2E tests
make test-e2e  

# Benchmarks
make bench

# Coverage
make test-coverage
```

### Quality Checks

```bash
# Static analysis + tests + coverage
make quality-gate

# CI pipeline
make ci
```

## 🎯 Design Philosophy

- **Test-Driven Development**: RED → GREEN → REFACTOR
- **High Precision**: Minimize false positives with escape analysis
- **High Performance**: Efficient AST cache and rule cache optimization
- **Extensibility**: Pluggable rule engine
- **Integration**: Compatibility with existing toolchains

## 🏛️ Architecture

```
├── cmd/gcpclosecheck/          # CLI entry point
├── internal/
│   ├── analyzer/               # Analysis engine
│   │   ├── analyzer.go         # Main analyzer
│   │   ├── resource_tracker.go # Resource tracking
│   │   ├── defer_analyzer.go   # defer statement analysis
│   │   ├── context_analyzer.go # context analysis
│   │   └── escape_analyzer.go  # Escape analysis
│   └── config/                 # Configuration management
├── testdata/                   # E2E test data
└── rules/                      # Default rules
```

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

### Development Guidelines

- Test-driven development with TDD
- Quality checks with golangci-lint
- Maintain 80%+ test coverage
- Prevent performance regressions

## 📄 License

MIT License - See [LICENSE](LICENSE) file for details.

## 🙋 Support

- **Issues**: [GitHub Issues](https://github.com/yukia3e/gcpclosecheck/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yukia3e/gcpclosecheck/discussions)
