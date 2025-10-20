terraform {
  required_providers {
    terraprobe = {
      source  = "hashicorp/terraprobe"
      version = "0.1.0"
    }
  }
}

provider "terraprobe" {
  default_timeout     = 10
  default_retries     = 2
  default_retry_delay = 3
}

# ============================================================================
# HTTP Tests - Using httpbin.org for comprehensive HTTP testing
# ============================================================================

resource "terraprobe_http_test" "httpbin_get" {
  name               = "HTTPBin GET Request"
  url                = "https://httpbin.org/get"
  method             = "GET"
  expect_status_code = 200
  expect_contains    = "args"
}

resource "terraprobe_http_test" "httpbin_json" {
  name               = "HTTPBin JSON Response"
  url                = "https://httpbin.org/json"
  method             = "GET"
  expect_status_code = 200
  expect_contains    = "slideshow"
}

resource "terraprobe_http_test" "httpbin_user_agent" {
  name               = "HTTPBin User Agent Check"
  url                = "https://httpbin.org/user-agent"
  method             = "GET"
  expect_status_code = 200
  expect_contains    = "TerraProbe"

  headers = {
    "User-Agent" = "TerraProbe-Test/1.0"
  }
}

resource "terraprobe_http_test" "github_api" {
  name               = "GitHub API Status"
  url                = "https://api.github.com/status"
  method             = "GET"
  expect_status_code = 200
}

# ============================================================================
# TCP Tests - Testing various public services
# ============================================================================

resource "terraprobe_tcp_test" "google_dns" {
  name = "Google Public DNS"
  host = "8.8.8.8"
  port = 53
}

resource "terraprobe_tcp_test" "cloudflare_dns" {
  name = "Cloudflare DNS"
  host = "1.1.1.1"
  port = 53
}

resource "terraprobe_tcp_test" "google_https" {
  name = "Google HTTPS"
  host = "google.com"
  port = 443
}

# ============================================================================
# DNS Tests - Testing various record types
# ============================================================================

resource "terraprobe_dns_test" "google_a_record" {
  name        = "Google.com A Record"
  hostname    = "google.com"
  record_type = "A"
}

resource "terraprobe_dns_test" "gmail_mx_record" {
  name        = "Gmail MX Records"
  hostname    = "gmail.com"
  record_type = "MX"
}

resource "terraprobe_dns_test" "google_ipv6" {
  name        = "Google.com AAAA Record (IPv6)"
  hostname    = "google.com"
  record_type = "AAAA"
}

# PostgreSQL database test
# This will use Docker containers in unit tests
# For integration tests, it tries to connect to localhost
resource "terraprobe_db_test" "postgres_test" {
  name          = "PostgreSQL Test"
  type          = "postgres"
  host          = "localhost"
  port          = 5432
  username      = "postgres"
  password      = "postgres"
  database      = "postgres"
  ssl_mode      = "disable"
  query         = "SELECT 1"
  timeout       = 5
  retries       = 2
  retry_delay   = 1
}

# MySQL database test
# This will use Docker containers in unit tests
# For integration tests, it tries to connect to localhost
resource "terraprobe_db_test" "mysql_test" {
  name          = "MySQL Test"
  type          = "mysql"
  host          = "localhost"
  port          = 3306
  username      = "root"
  password      = "mysql"
  database      = "mysql"
  query         = "SELECT 1"
  timeout       = 5
  retries       = 2
  retry_delay   = 1
}

# ============================================================================
# Test Suites - Organizing tests by category
# ============================================================================

resource "terraprobe_test_suite" "http_suite" {
  name        = "HTTP Test Suite"
  description = "All HTTP tests using httpbin.org and public APIs"

  http_tests = [
    terraprobe_http_test.httpbin_get.id,
    terraprobe_http_test.httpbin_json.id,
    terraprobe_http_test.httpbin_user_agent.id,
    terraprobe_http_test.github_api.id
  ]
}

resource "terraprobe_test_suite" "network_suite" {
  name        = "Network Test Suite"
  description = "TCP and DNS connectivity tests"

  tcp_tests = [
    terraprobe_tcp_test.google_dns.id,
    terraprobe_tcp_test.cloudflare_dns.id,
    terraprobe_tcp_test.google_https.id
  ]

  dns_tests = [
    terraprobe_dns_test.google_a_record.id,
    terraprobe_dns_test.gmail_mx_record.id,
    terraprobe_dns_test.google_ipv6.id
  ]
}

resource "terraprobe_test_suite" "all_tests" {
  name        = "Complete Test Suite"
  description = "All infrastructure tests (HTTP, TCP, DNS, Database)"

  http_tests = [
    terraprobe_http_test.httpbin_get.id,
    terraprobe_http_test.httpbin_json.id,
    terraprobe_http_test.httpbin_user_agent.id,
    terraprobe_http_test.github_api.id
  ]

  tcp_tests = [
    terraprobe_tcp_test.google_dns.id,
    terraprobe_tcp_test.cloudflare_dns.id,
    terraprobe_tcp_test.google_https.id
  ]

  dns_tests = [
    terraprobe_dns_test.google_a_record.id,
    terraprobe_dns_test.gmail_mx_record.id,
    terraprobe_dns_test.google_ipv6.id
  ]

  db_tests = [
    terraprobe_db_test.postgres_test.id,
    terraprobe_db_test.mysql_test.id
  ]
}

# ============================================================================
# Outputs
# ============================================================================

output "http_suite_summary" {
  value = {
    all_passed   = terraprobe_test_suite.http_suite.all_passed
    passed_count = terraprobe_test_suite.http_suite.passed_count
    failed_count = terraprobe_test_suite.http_suite.failed_count
    total_count  = terraprobe_test_suite.http_suite.total_count
  }
}

output "network_suite_summary" {
  value = {
    all_passed   = terraprobe_test_suite.network_suite.all_passed
    passed_count = terraprobe_test_suite.network_suite.passed_count
    failed_count = terraprobe_test_suite.network_suite.failed_count
    total_count  = terraprobe_test_suite.network_suite.total_count
  }
}

output "httpbin_tests" {
  description = "HTTPBin.org test results"
  value = {
    get_test = {
      passed        = terraprobe_http_test.httpbin_get.test_passed
      status_code   = terraprobe_http_test.httpbin_get.last_status_code
      response_time = terraprobe_http_test.httpbin_get.last_response_time
    }
    json_test = {
      passed        = terraprobe_http_test.httpbin_json.test_passed
      status_code   = terraprobe_http_test.httpbin_json.last_status_code
      response_time = terraprobe_http_test.httpbin_json.last_response_time
    }
    user_agent_test = {
      passed        = terraprobe_http_test.httpbin_user_agent.test_passed
      status_code   = terraprobe_http_test.httpbin_user_agent.last_status_code
      response_time = terraprobe_http_test.httpbin_user_agent.last_response_time
    }
  }
}

output "dns_tests" {
  description = "DNS resolution test results"
  value = {
    google_ipv4 = {
      passed  = terraprobe_dns_test.google_a_record.test_passed
      results = terraprobe_dns_test.google_a_record.last_result
      time_ms = terraprobe_dns_test.google_a_record.last_result_time
    }
    google_ipv6 = {
      passed  = terraprobe_dns_test.google_ipv6.test_passed
      results = terraprobe_dns_test.google_ipv6.last_result
      time_ms = terraprobe_dns_test.google_ipv6.last_result_time
    }
    gmail_mx = {
      passed  = terraprobe_dns_test.gmail_mx_record.test_passed
      results = terraprobe_dns_test.gmail_mx_record.last_result
      time_ms = terraprobe_dns_test.gmail_mx_record.last_result_time
    }
  }
}

output "tcp_tests" {
  description = "TCP connectivity test results"
  value = {
    google_dns = {
      passed       = terraprobe_tcp_test.google_dns.test_passed
      connect_time = terraprobe_tcp_test.google_dns.last_connect_time
    }
    cloudflare_dns = {
      passed       = terraprobe_tcp_test.cloudflare_dns.test_passed
      connect_time = terraprobe_tcp_test.cloudflare_dns.last_connect_time
    }
    google_https = {
      passed       = terraprobe_tcp_test.google_https.test_passed
      connect_time = terraprobe_tcp_test.google_https.last_connect_time
    }
  }
}

output "postgres_test_results" {
  value = {
    passed       = terraprobe_db_test.postgres_test.test_passed
    rows         = terraprobe_db_test.postgres_test.last_result_rows
    query_time   = terraprobe_db_test.postgres_test.last_query_time
    error        = terraprobe_db_test.postgres_test.error
  }
}

output "mysql_test_results" {
  value = {
    passed       = terraprobe_db_test.mysql_test.test_passed
    rows         = terraprobe_db_test.mysql_test.last_result_rows
    query_time   = terraprobe_db_test.mysql_test.last_query_time
    error        = terraprobe_db_test.mysql_test.error
  }
}

output "overall_summary" {
  description = "Overall test suite summary"
  value = {
    all_passed   = terraprobe_test_suite.all_tests.all_passed
    passed_count = terraprobe_test_suite.all_tests.passed_count
    failed_count = terraprobe_test_suite.all_tests.failed_count
    total_count  = terraprobe_test_suite.all_tests.total_count
  }
} 