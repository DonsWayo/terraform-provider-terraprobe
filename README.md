# TerraProbe

[![Go Version](https://img.shields.io/github/go-mod/go-version/DonsWayo/terraprobe?style=flat-square)](https://golang.org/doc/go1.25)
[![Terraform Registry](https://img.shields.io/badge/registry-terraform.io-blue?style=flat-square)](https://registry.terraform.io/namespaces/DonsWayo)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg?style=flat-square)](https://opensource.org/licenses/MPL-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/DonsWayo/terraprobe?style=flat-square)](https://goreportcard.com/report/github.com/DonsWayo/terraprobe)
[![GitHub release](https://img.shields.io/github/v/release/DonsWayo/terraprobe?style=flat-square)](https://github.com/DonsWayo/terraprobe/releases)

Infrastructure testing Terraform provider that validates your deployed resources are functioning correctly.

## What is TerraProbe?

TerraProbe lets you write infrastructure tests directly in Terraform. After deploying your infrastructure, you can verify that HTTP endpoints respond correctly, TCP ports are accessible, DNS records resolve properly, and databases are reachable.

## Features

- **HTTP Testing**: Validate API endpoints, check status codes, verify response content
- **TCP Testing**: Ensure services are listening on expected ports
- **DNS Testing**: Verify domain resolution for A, AAAA, CNAME, MX, TXT, and NS records
- **Database Testing**: Test PostgreSQL and MySQL connectivity and run validation queries
- **Test Suites**: Group related tests and get aggregated results
- **Retry Logic**: Built-in retry mechanisms for handling transient failures

## Installation

### Using Terraform Registry

```hcl
terraform {
  required_providers {
    terraprobe = {
      source  = "DonsWayo/terraprobe"
      version = "~> 0.1"
    }
  }
}

provider "terraprobe" {
  default_timeout     = 10
  default_retries     = 3
  default_retry_delay = 5
}
```

### Building from Source

```bash
git clone https://github.com/DonsWayo/terraprobe.git
cd terraprobe

# Using Task (recommended)
task install

# Or manually
go build -o terraform-provider-terraprobe
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/DonsWayo/terraprobe/0.1.0/$(go env GOOS)_$(go env GOARCH)/
cp terraform-provider-terraprobe ~/.terraform.d/plugins/registry.terraform.io/DonsWayo/terraprobe/0.1.0/$(go env GOOS)_$(go env GOARCH)/
```

## Quick Start

Here's a simple example that tests an API endpoint, database connection, and DNS resolution:

```hcl
# Test API endpoint
resource "terraprobe_http_test" "api" {
  name               = "API Health Check"
  url                = "https://api.example.com/health"
  expect_status_code = 200
  expect_contains    = "healthy"
}

# Test database port
resource "terraprobe_tcp_test" "database" {
  name = "PostgreSQL Port"
  host = "db.example.com"
  port = 5432
}

# Test DNS resolution
resource "terraprobe_dns_test" "website" {
  name        = "Website DNS"
  hostname    = "www.example.com"
  record_type = "A"
}

# Group tests together
resource "terraprobe_test_suite" "production" {
  name        = "Production Tests"
  description = "All production environment tests"

  http_tests = [terraprobe_http_test.api.id]
  tcp_tests  = [terraprobe_tcp_test.database.id]
  dns_tests  = [terraprobe_dns_test.website.id]
}

# Check results
output "test_results" {
  value = {
    passed = terraprobe_test_suite.production.all_passed
    summary = "${terraprobe_test_suite.production.passed_count}/${terraprobe_test_suite.production.total_count} tests passed"
  }
}
```

## Resource Documentation

### HTTP Test

Tests HTTP/HTTPS endpoints with various validation options.

```hcl
resource "terraprobe_http_test" "example" {
  name               = "API Test"
  url                = "https://api.example.com/v1/status"
  method             = "GET"                    # GET, POST, PUT, DELETE, etc.
  expect_status_code = 200
  expect_contains    = "operational"            # Optional: string to find in response
  timeout            = 30                       # Optional: override default timeout
  retries            = 5                        # Optional: override default retries

  headers = {
    "Authorization" = "Bearer ${var.api_token}"
    "Content-Type"  = "application/json"
  }

  body = jsonencode({                          # Optional: request body
    test = true
  })
}
```

### TCP Test

Verifies TCP connectivity to services.

```hcl
resource "terraprobe_tcp_test" "example" {
  name    = "Redis Connection"
  host    = "cache.example.com"
  port    = 6379
  timeout = 10
  retries = 3
}
```

### DNS Test

Validates DNS resolution and record values.

```hcl
resource "terraprobe_dns_test" "example" {
  name          = "Mail Server DNS"
  hostname      = "mail.example.com"
  record_type   = "MX"                    # A, AAAA, CNAME, MX, TXT, NS
  expect_result = "10 mail.example.com."  # Optional: expected value
  resolver      = "1.1.1.1"               # Optional: custom DNS server
  timeout       = 5
}
```

### Database Test

Tests database connectivity and executes validation queries.

```hcl
resource "terraprobe_db_test" "postgres" {
  name     = "PostgreSQL Test"
  type     = "postgres"              # postgres or mysql
  host     = "db.example.com"
  port     = 5432
  database = "myapp"
  username = var.db_user
  password = var.db_password
  ssl_mode = "require"               # disable, require, verify-ca, verify-full

  query = "SELECT COUNT(*) FROM users WHERE active = true"

  # Connection pool settings
  max_open_conn = 25
  max_idle_conn = 5
  max_lifetime  = 300
}

resource "terraprobe_db_test" "mysql" {
  name     = "MySQL Test"
  type     = "mysql"
  host     = "mysql.example.com"
  port     = 3306
  database = "myapp"
  username = var.mysql_user
  password = var.mysql_password

  query = "SELECT VERSION()"
}
```

### Test Suite

Groups multiple tests for organized results.

```hcl
resource "terraprobe_test_suite" "production" {
  name        = "Production Environment"
  description = "All production tests"

  http_tests = [
    terraprobe_http_test.api.id,
    terraprobe_http_test.website.id,
  ]

  tcp_tests = [
    terraprobe_tcp_test.database.id,
    terraprobe_tcp_test.cache.id,
  ]

  dns_tests = [
    terraprobe_dns_test.api.id,
    terraprobe_dns_test.cdn.id,
  ]

  db_tests = [
    terraprobe_db_test.postgres.id,
    terraprobe_db_test.mysql.id,
  ]
}
```

## Output Attributes

All test resources provide these attributes:

- `test_passed` - Boolean indicating if the test passed
- `last_run` - Timestamp of last test execution
- `error` - Error message if test failed

Additional attributes by test type:
- HTTP: `last_response_time`, `last_status_code`, `last_response_body`
- TCP: `last_connect_time`
- DNS: `last_result`, `last_result_time`
- Database: `last_query_time`, `last_result_rows`

## Development

### Requirements

- Go 1.25+
- Terraform 1.13+ or OpenTofu 1.10+
- Docker (for database tests)
- [Task](https://taskfile.dev) (optional, for automation)

### Building

```bash
# Using Task
task build

# Manually
go build -o terraform-provider-terraprobe
```

### Testing

```bash
# All tests with Task
task test:all

# Unit tests only
task test:unit

# Integration tests (requires Docker)
task test:integration

# Manual testing
go test -v ./internal/provider/...
```

### Task Commands

```bash
task --list                  # List all available tasks
task build                   # Build the provider
task install                 # Build and install locally
task test:unit              # Run unit tests
task test:integration       # Run integration tests with databases
task lint                   # Run linters
task fmt                    # Format code
task db:start               # Start test database containers
task db:stop                # Stop database containers
task clean                  # Clean build artifacts
```

## Provider Configuration

```hcl
provider "terraprobe" {
  default_timeout     = 10    # Default timeout in seconds for all tests
  default_retries     = 3     # Number of retry attempts
  default_retry_delay = 5     # Seconds between retries
  user_agent          = "TerraProbe/1.0"  # User agent for HTTP tests
}
```

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/new-feature`)
3. Commit your changes (`git commit -m 'Add new feature'`)
4. Push to the branch (`git push origin feature/new-feature`)
5. Create a Pull Request

We use conventional commits:
- `feat:` New features
- `fix:` Bug fixes
- `docs:` Documentation changes
- `test:` Test additions or fixes
- `chore:` Maintenance tasks

## License

This project is licensed under the Mozilla Public License 2.0 - Copyright (c) 2025 Juan Jose Caracedo - see the [LICENSE](LICENSE) file for details.

## Support

- [Issues](https://github.com/DonsWayo/terraprobe/issues)
- [Discussions](https://github.com/DonsWayo/terraprobe/discussions)
- [Wiki](https://github.com/DonsWayo/terraprobe/wiki)