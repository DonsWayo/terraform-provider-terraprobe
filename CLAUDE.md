# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TerraProbe is a Terraform provider that enables infrastructure testing directly within Terraform workflows. It provides resources for testing HTTP endpoints, TCP connections, DNS resolution, and database connectivity.

## Key Architecture Components

### Provider Structure
- **Main Provider**: `internal/provider/provider.go:23` - Implements the core TerraProbeProvider with configuration for timeouts, retries, and HTTP client management
- **Resource Registration**: Resources are registered in `internal/provider/provider.go:123-130` via the Resources() method
- **Client Configuration**: Shared client config (`TerraProbeClientConfig`) at `internal/provider/provider.go:116` manages HTTP client, retries, and user agent across all resources

### Test Resources
Each test resource follows a consistent pattern with Create/Read/Update/Delete lifecycle:
- **HTTP Test**: `internal/provider/http_test_resource.go` - Tests HTTP/HTTPS endpoints with status code and body content validation
- **TCP Test**: `internal/provider/tcp_test_resource.go` - Tests TCP connectivity to specified hosts and ports
- **DNS Test**: `internal/provider/dns_test_resource.go` - Tests DNS resolution for various record types (A, AAAA, CNAME, MX, TXT, NS)
- **Database Test**: `internal/provider/db_test_resource.go` - Tests database connectivity for PostgreSQL and MySQL with query execution
- **Test Suite**: `internal/provider/test_suite_resource.go` - Groups multiple tests together for aggregated results

### Testing Infrastructure
Tests use Docker containers for database testing to ensure isolation and consistency. The test setup automatically manages container lifecycle during test execution.

## Common Development Commands

### Build and Installation
```bash
# Build the provider
make build
# or
go build -o terraform-provider-terraprobe

# Install locally for testing
make install

# Manual local installation (for macOS)
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/hashicorp/terraprobe/0.1.0/darwin_amd64/
cp terraform-provider-terraprobe ~/.terraform.d/plugins/registry.terraform.io/hashicorp/terraprobe/0.1.0/darwin_amd64/
```

### Testing
```bash
# Run all unit tests with Docker for database tests
make test
# or
go test -v -cover -timeout=120s -parallel=10 ./...

# Run a single test
go test -v -run TestHttpTestResource ./internal/provider/

# Run acceptance tests (requires TF_ACC=1)
make testacc
# or
TF_ACC=1 go test -v -cover -timeout 120m ./...

# Run the test configuration script (from root directory)
cd test-config
./run-test.sh --all    # Run both unit tests and Terraform apply
./run-test.sh --unit   # Run only unit tests
./run-test.sh --test   # Run only Terraform apply
```

### Code Quality
```bash
# Format code
make fmt
# or
gofmt -s -w -e .

# Run linters
make lint
# or
golangci-lint run

# Generate documentation
make generate
```

## Resource Implementation Pattern

When implementing new test resources:

1. **Resource File Structure**: Create `internal/provider/{resource_name}_test_resource.go` implementing the resource.Resource interface
2. **Model Definition**: Define a model struct with `tfsdk` tags for Terraform schema mapping
3. **Schema Method**: Define resource attributes including name, test parameters, and result fields (test_passed, last_run, error)
4. **Create Method**: Implement test execution logic with retry mechanism using provider-level configuration
5. **Read Method**: Refresh resource state from Terraform state (typically no-op for test resources)
6. **Update Method**: Re-run tests when configuration changes
7. **Delete Method**: Remove resource from state (no cleanup needed for test resources)
8. **Test File**: Create corresponding `_test.go` file with unit and acceptance tests

## Provider Configuration Flow

1. Provider receives configuration in `Configure()` method at `internal/provider/provider.go:68`
2. Default values are set for timeout (30s), retries (3), retry delay (5s)
3. HTTP client is created with specified timeout
4. `TerraProbeClientConfig` is passed to resources via ResourceData
5. Resources retrieve configuration during Configure() lifecycle

## Docker Requirements for Development

Database tests require Docker to be installed and running. Tests automatically:
- Detect Docker availability
- Spin up PostgreSQL/MySQL containers as needed
- Clean up containers after test completion
- Fall back gracefully if Docker is unavailable

## Testing Patterns

- Unit tests use mock servers for HTTP/TCP testing
- Database tests use Docker containers with testcontainers-go
- Test files follow `*_test.go` naming convention
- Acceptance tests require `TF_ACC=1` environment variable
- Tests run in parallel with `-parallel=10` flag for efficiency