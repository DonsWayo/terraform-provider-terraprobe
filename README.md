# TerraProbe - Infrastructure Testing Terraform Provider

TerraProbe is a Terraform provider that facilitates infrastructure testing directly through Terraform. It allows you to define and run automated tests for HTTP endpoints, TCP connections, DNS resolution, and more, integrating testing directly into your Terraform workflow.

## Features

- **HTTP Testing**: Test HTTP/HTTPS endpoints for status codes, response body content, and more
- **TCP Testing**: Test TCP connections to specified hosts and ports
- **DNS Testing**: Test DNS resolution for various record types (A, AAAA, CNAME, MX, TXT, NS)
- **Test Suites**: Group tests together and get aggregated results
- **Terraform Integration**: Seamlessly integrates with your existing Terraform workflows

## Resources

The TerraProbe provider includes the following resources:

- `terraprobe_http_test` - Test HTTP endpoints
- `terraprobe_tcp_test` - Test TCP connectivity
- `terraprobe_dns_test` - Test DNS resolution
- `terraprobe_db_test` - Test database connectivity
- `terraprobe_test_suite` - Group multiple tests together

## Usage

### Provider Configuration

```hcl
provider "terraprobe" {
  retries = 3           # Number of retries for tests that fail (default: 3)
  retry_delay = 5       # Delay between retries in seconds (default: 5)
}
```

### HTTP Test

```hcl
resource "terraprobe_http_test" "example" {
  name             = "API Health Check"
  url              = "https://api.example.com/health"
  method           = "GET"
  expect_status_code = 200
  expect_contains  = "healthy"
  headers = {
    "User-Agent" = "TerraProbe"
    "Authorization" = "Bearer ${var.api_token}"
  }
}

output "api_status" {
  value = terraprobe_http_test.example.test_passed ? "Healthy" : "Unhealthy"
}
```

### TCP Test

```hcl
resource "terraprobe_tcp_test" "example" {
  name = "Database Connection"
  host = "db.example.com"
  port = 5432
}

output "db_status" {
  value = terraprobe_tcp_test.example.test_passed ? "Connected" : "Failed"
}
```

### DNS Test

```hcl
resource "terraprobe_dns_test" "example" {
  name        = "Domain Resolution Check"
  hostname    = "example.com"
  record_type = "A"            # Supports A, AAAA, CNAME, MX, TXT, NS
  
  # Optional: Specify an expected result
  expect_result = "93.184.216.34"
  
  # Optional: Use a specific DNS resolver
  resolver = "8.8.8.8"
}

output "dns_status" {
  value = {
    resolved = terraprobe_dns_test.example.test_passed
    ip_addresses = terraprobe_dns_test.example.last_result
    query_time_ms = terraprobe_dns_test.example.last_result_time
  }
}
```

### Test Suite

```hcl
resource "terraprobe_test_suite" "all_tests" {
  name = "System Health Checks"
  description = "Tests for all critical system components"
  
  http_tests = [
    terraprobe_http_test.api.id,
    terraprobe_http_test.website.id
  ]
  
  tcp_tests = [
    terraprobe_tcp_test.database.id,
    terraprobe_tcp_test.redis.id
  ]
}

output "system_health" {
  value = {
    passing = terraprobe_test_suite.all_tests.passed_count
    failing = terraprobe_test_suite.all_tests.failed_count
    all_healthy = terraprobe_test_suite.all_tests.all_passed
  }
}
```

### Database Test Resource

The database test resource (`terraprobe_db_test`) allows you to test connectivity and query execution against various database engines.

```hcl
resource "terraprobe_db_test" "postgres_test" {
  name     = "PostgreSQL Database Test"
  type     = "postgres"  # Supported types: postgres, mysql
  host     = "db.example.com"
  port     = 5432
  username = "dbuser"
  password = "dbpassword"
  database = "mydb"
  query    = "SELECT 1"  # Optional query to execute
  ssl_mode = "disable"   # For PostgreSQL: disable, require, verify-ca, verify-full
  
  # Connection pool settings (optional)
  max_lifetime = 1800    # Maximum connection lifetime in seconds
  max_idle_conn = 10     # Maximum idle connections in the pool
  max_open_conn = 100    # Maximum open connections
  
  # Retry settings (optional)
  timeout    = 10
  retries    = 3
  retry_delay = 5
}
```

The resource provides the following attributes:

- `test_passed`: Boolean indicating if the test passed
- `last_run`: Timestamp of the last test run
- `last_query_time`: Duration in milliseconds the query took to execute
- `last_result_rows`: Number of rows returned by the query
- `error`: Error message if the test failed

## Development

### Requirements

- [Go](https://golang.org/doc/install) 1.18+ (to build the provider plugin)
- [Terraform](https://www.terraform.io/downloads.html) 0.14.x+

### Building the Provider

1. Clone the repository
2. Build the provider
   ```shell
   go build -o terraform-provider-terraprobe
   ```
3. Install the provider locally (for testing)
   ```shell
   mkdir -p ~/.terraform.d/plugins/registry.terraform.io/hashicorp/terraprobe/0.1.0/$(go env GOOS)_$(go env GOARCH)/
   cp terraform-provider-terraprobe ~/.terraform.d/plugins/registry.terraform.io/hashicorp/terraprobe/0.1.0/$(go env GOOS)_$(go env GOARCH)/
   ```

### Testing

#### Run Unit Tests

```shell
go test ./internal/provider/... -v
```

#### Run the Test Configuration

```shell
# From the root directory
cd test-config
./run-test.sh --all
```

The `run-test.sh` script supports the following options:
- `--test`: Run Terraform to test the provider
- `--unit`: Run unit tests only
- `--all`: Run both unit tests and Terraform tests

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the Mozilla Public License 2.0 - see the LICENSE file for details.
