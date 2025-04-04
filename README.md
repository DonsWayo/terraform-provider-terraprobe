# TerraProbe: Terraform Provider for Infrastructure Testing

TerraProbe is a Terraform provider that validates infrastructure after deployment through various tests. It integrates with the regular Terraform workflow, runs post-deployment tests against your infrastructure, and saves test results as Terraform state.

## Features

- **Seamless Integration**: Works within your existing Terraform workflow
- **HTTP Testing**: Validate REST APIs, web services, and HTTP endpoints
- **TCP Testing**: Verify network connectivity to hosts and ports
- **Test Results in State**: All test results are saved in Terraform state for easy access
- **Configurable Retries**: Configure timeout, retry count, and retry delay
- **Rich Output**: Get detailed test results including response times, status codes, and error messages

## Usage

```hcl
terraform {
  required_providers {
    terraprobe = {
      source = "hashicorp/terraprobe"
    }
  }
}

provider "terraprobe" {
  default_timeout     = 30
  default_retries     = 3
  default_retry_delay = 5
}

# Define an HTTP test to validate an API endpoint
resource "terraprobe_http_test" "api_health" {
  name        = "API Health Check"
  url         = "https://${aws_lb.api.dns_name}/health"
  method      = "GET"
  
  # Define assertions
  expect_status_code = 200
  expect_contains   = "status: healthy"
}

# Define a TCP test to verify database connectivity
resource "terraprobe_tcp_test" "database_connection" {
  name = "Database Connection Test"
  host = aws_db_instance.postgres.address
  port = aws_db_instance.postgres.port
  
  # This ensures the test runs after the database is created
  depends_on = [aws_db_instance.postgres]
}

# Output test results
output "infrastructure_tests" {
  value = {
    api_status = {
      passed      = terraprobe_http_test.api_health.test_passed
      status_code = terraprobe_http_test.api_health.last_status_code
      error       = terraprobe_http_test.api_health.error
    }
    db_connection = {
      passed = terraprobe_tcp_test.database_connection.test_passed
      error  = terraprobe_tcp_test.database_connection.error
    }
  }
}
```

## Provider Configuration

| Argument | Description | Default |
|----------|-------------|---------|
| `default_timeout` | Default timeout in seconds for all tests | `30` |
| `default_retries` | Default number of retries for all tests | `3` |
| `default_retry_delay` | Default delay between retries in seconds | `5` |
| `user_agent` | User agent to use for HTTP requests | `TerraProbe Terraform Provider` |

## Resources

### `terraprobe_http_test`

The HTTP test resource allows you to validate HTTP endpoints.

#### Arguments

| Argument | Description | Required |
|----------|-------------|----------|
| `name` | Descriptive name for the test | Yes |
| `url` | URL to test | Yes |
| `method` | HTTP method (GET, POST, PUT, DELETE, etc.) | No (default: GET) |
| `headers` | Map of HTTP headers | No |
| `body` | Request body for POST, PUT, etc. | No |
| `timeout` | Timeout in seconds (overrides provider default) | No |
| `retries` | Number of retries (overrides provider default) | No |
| `retry_delay` | Delay between retries in seconds (overrides provider default) | No |
| `expect_status_code` | Expected HTTP status code | No (default: 200) |
| `expect_contains` | String to look for in the response body | No |

#### Attributes

| Attribute | Description |
|-----------|-------------|
| `id` | Test identifier |
| `last_run` | Timestamp of the last test run |
| `last_status_code` | Status code from the last test run |
| `last_response_body` | Response body from the last test run |
| `last_response_time` | Response time in milliseconds from the last test run |
| `test_passed` | Whether the test passed |
| `error` | Error message if the test failed |

### `terraprobe_tcp_test`

The TCP test resource allows you to validate TCP connectivity to hosts and ports.

#### Arguments

| Argument | Description | Required |
|----------|-------------|----------|
| `name` | Descriptive name for the test | Yes |
| `host` | Host to connect to (IP address or hostname) | Yes |
| `port` | Port to connect to | Yes |
| `timeout` | Timeout in seconds (overrides provider default) | No |
| `retries` | Number of retries (overrides provider default) | No |
| `retry_delay` | Delay between retries in seconds (overrides provider default) | No |

#### Attributes

| Attribute | Description |
|-----------|-------------|
| `id` | Test identifier |
| `last_run` | Timestamp of the last test run |
| `last_connect_time` | Connection time in milliseconds from the last test run |
| `test_passed` | Whether the test passed (connection was established) |
| `error` | Error message if the test failed |

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.1
- [Go](https://golang.org/doc/install) >= 1.20

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Future Plans

- Additional test types (gRPC, DNS, Kubernetes, Databases)
- Test suites for grouping tests
- AI-powered test result analysis
- Customizable notifications and actions on test failure
