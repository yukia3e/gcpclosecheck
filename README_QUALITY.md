# Quality Check Tools Usage Guide

A comprehensive guide on how to use the tool suite for analyzing and improving the quality and performance of the gcpclosecheck project.

[日本語版](README_QUALITY.ja.md) | English

## 📋 Overview

This tool suite provides the following features:

- **Test Analysis**: Coverage analysis, detailed evaluation of test execution results
- **Code Quality Verification**: Static analysis, security scans, build verification
- **Performance Measurement**: Benchmark execution, profiling
- **Automatic Fixes**: Automatic fixing of detected issues and code formatting
- **Report Generation**: Integrated reports and detailed analysis results
- **Continuous Improvement Support**: Quality metrics history management and trend analysis

## 🚀 Quick Start

### Prerequisites

- Go 1.25.0 or higher
- golangci-lint 2.4.0 or higher (recommended)
- gosec, govulncheck (optional)

### Basic Usage

```bash
# Run all analyses
scripts/quality-check.sh

# Test analysis only
scripts/quality-check.sh --test

# Quality verification only
scripts/quality-check.sh --quality

# Performance measurement only
scripts/quality-check.sh --perf
```

## 📚 Detailed Usage of Each Script

### quality-check.sh - Main Orchestration Script

Executes comprehensive quality and performance analysis of the project.

**Usage:**
```bash
scripts/quality-check.sh [options]
```

**Options:**
- `--test`: Execute test analysis only (coverage, test results)
- `--quality`: Execute quality verification only (static analysis, security)  
- `--perf`: Execute performance measurement only (benchmarks, profiling)
- `--all`: Execute all analyses (default)
- `--help`: Show help

**Output Files:**
- `reports/performance_summary.json`: Execution time and performance information

### test-analysis.sh - Dedicated Test Analysis Script

Performs detailed analysis of test coverage and evaluation of test execution results.

**Main Features:**
- Coverage measurement using go test -cover
- Detailed analysis of test execution results
- HTML coverage report generation
- Detailed report creation for failed tests

**Output Files:**
- `reports/test_results.json`: Structured test results
- `reports/test_summary.txt`: Test results summary
- `reports/coverage.html`: HTML coverage report
- `reports/failed_tests_detail.txt`: Details of failed tests

### code-quality.sh - Code Quality Verification Script

Executes static analysis, security scans, and build verification.

**Main Features:**
- Static analysis with golangci-lint
- Security scanning with gosec
- Vulnerability checking with govulncheck
- Build verification with go build and go vet

**Output Files:**
- `reports/lint_results.json`: Static analysis results
- `reports/security_results.json`: Security scan results
- `reports/build_results.json`: Build verification results

### performance-check.sh - Performance Measurement Script

Executes benchmark testing and profiling analysis.

**Main Features:**
- Benchmark execution with go test -bench
- CPU and memory profiling
- Performance metrics calculation

**Output Files:**
- `reports/benchmark_results.json`: Benchmark results
- `reports/profile_results.json`: Profiling results

### fix-issues.sh - Automatic Fix Script

Performs automatic fixing and prioritization of detected issues.

**Main Features:**
- Automatic formatting with go fmt
- Import organization with goimports
- Automatic fixes with golangci-lint --fix
- Issue prioritization and recommended fix order suggestions

**Output Files:**
- `reports/fix_results.json`: Fix execution results
- `reports/priority_results.json`: Issue priority analysis

### generate-report.sh - Report Generation Script

Generates integrated reports and detailed analysis reports.

**Main Features:**
- Integrated report generation from all analysis results
- Detailed technical analysis report creation
- Executive summary report generation for management

**Output Files:**
- `reports/integrated_report.md`: Integrated report
- `reports/detailed_report.md`: Detailed technical report
- `reports/executive_summary.md`: Executive summary

### track-progress.sh - Continuous Improvement Support Script

Manages quality metrics history and performs trend analysis.

**Usage:**
```bash
scripts/track-progress.sh [options]
```

**Options:**
- `--track`: Execute quality metrics tracking
- `--trend`: Execute trend analysis only
- `--compare`: Execute comparison analysis only

**Output Files:**
- `reports/history/`: Quality metrics history data
- `reports/trend_analysis.md`: Trend analysis report
- `reports/progress_report.md`: Progress report

## 🎯 Best Practices

### 1. Recommended Regular Execution

```bash
# Recommended weekly quality check execution
# Add to crontab:
# 0 2 * * 1 /path/to/gcpclosecheck/scripts/quality-check.sh
```

### 2. Quality Improvement Workflow

1. **Analysis Execution**: Understand current state with `scripts/quality-check.sh`
2. **Issue Identification**: Check reports in `reports/` directory
3. **Automatic Fixes**: Resolve fixable issues with `scripts/fix-issues.sh`
4. **Manual Fixes**: Address remaining issues in priority order
5. **Effect Measurement**: Confirm improvement effects with `scripts/track-progress.sh`

### 3. CI/CD Pipeline Integration

```yaml
# GitHub Actions example
- name: Quality Check
  run: |
    chmod +x scripts/quality-check.sh
    scripts/quality-check.sh --test
```

### 4. Report Utilization Methods

- **Developers**: Check technical details in `reports/detailed_report.md`
- **Managers**: Understand overall situation with `reports/executive_summary.md`
- **Continuous Improvement**: Track improvement trends with `reports/trend_analysis.md`

## 🔧 Troubleshooting

### Common Issues and Solutions

#### 1. golangci-lint Configuration Error

**Error Example:**
```
Error: can't load config: unsupported version of the configuration
```

**Solution:**
```bash
# Update .golangci.yml
golangci-lint --version  # Check version
# Update to latest configuration format
```

#### 2. Mass Test Failures

**Solution Approach:**
1. Check failure causes in `reports/failed_tests_detail.txt`
2. Check for dependency issues: `go mod tidy`
3. Execute gradual test fixes

#### 3. Memory Shortage Error

**Solution:**
```bash
# For large projects, reduce parallelism
# Adjust MAX_PARALLEL_JOBS in scripts/quality-check.sh
```

#### 4. Unable to Obtain Coverage

**Check Items:**
- Do test files exist?
- Is go.mod configuration correct?
- Are tests actually executing?

### Known Issues

#### 1. GCP Library Dependency Errors

Some tests may encounter GCP client library import errors.

**Workaround:**
```bash
# Install necessary GCP libraries
go mod download cloud.google.com/go/...
```

#### 2. macOS date Command Compatibility

The date command behavior may differ between macOS and Linux.

**Already Addressed:**
The scripts automatically detect the environment and use appropriate commands.

## 📊 Operations Guide

### Operational Process for Continuous Quality Improvement

#### 1. Daily Monitoring

```bash
# For development teams: Daily quality checks
scripts/quality-check.sh --test
```

#### 2. Weekly Quality Review

```bash
# Full analysis and history comparison
scripts/quality-check.sh
scripts/track-progress.sh
```

#### 3. Monthly Quality Assessment

1. Create and share detailed reports
2. Quality metrics trend analysis
3. Review improvement plans

### Quality Gate Criteria

The following criteria are recommended:

- **Test Coverage**: 80% or higher
- **Test Failures**: 0 failures
- **Security Issues**: 0 issues
- **Critical Lint Issues**: 0 issues

### Maintenance

#### Regular Maintenance Tasks

```bash
# Weekly cleanup
scripts/cleanup.sh

# Monthly setup verification
scripts/setup.sh --verify
```

#### Log and Report Management

- Reports are automatically rotated
- Reports older than 30 days are automatically deleted
- History data is permanently stored in `reports/history/`

## 🔄 Update Guide

### Tool Updates

```bash
# Update Go toolchain
mise install go@latest

# Update golangci-lint
golangci-lint --version
# Install latest version
```

### Configuration Updates

- `.golangci.yml`: Adjust static analysis rules
- `scripts/utils.sh`: Change common settings
- `scripts/quality-check.sh`: Adjust execution parameters

## 📈 Understanding Quality Metrics

### Overall Quality Score Calculation Method

```
Overall Quality Score = Test Coverage(%) - (Failed Tests × 2) - (Lint Issues ÷ 2) - (Security Issues × 10)
```

### Recommended Improvement Order

1. **Security Issues** (Highest Priority): Fix immediately
2. **Test Failures**: Fix before release
3. **Coverage Deficiency**: Continuous improvement
4. **Lint Issues**: Fix in next release

## 🤝 Support

### Issue Reporting

Please report issues or improvement suggestions for quality check tools as project issues.

### Extension Methods

How to add new quality check features:

1. Create new script in `scripts/` directory
2. Integrate into `scripts/quality-check.sh`
3. Add tests to `scripts/tests/test-scripts.sh`
4. Update this documentation
