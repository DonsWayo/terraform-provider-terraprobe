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

# HTTP test for a reliable endpoint
resource "terraprobe_http_test" "github_status" {
  name  = "GitHub Status Check"
  url   = "https://www.githubstatus.com/"
  
  expect_status_code = 200
}

# TCP test for a reliable endpoint
resource "terraprobe_tcp_test" "google_dns" {
  name = "Google DNS Check"
  host = "8.8.8.8"
  port = 53
}

# DNS test for resolving a domain
resource "terraprobe_dns_test" "terraform_site" {
  name        = "Terraform DNS Check"
  hostname    = "terraform.io"
  record_type = "A"
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

# Test suite combining all tests
resource "terraprobe_test_suite" "all_tests" {
  name        = "All Tests"
  description = "Suite that runs all TerraProbe tests"
  
  http_tests = [
    terraprobe_http_test.github_status.id
  ]
  
  tcp_tests = [
    terraprobe_tcp_test.google_dns.id
  ]
  
  dns_tests = [
    terraprobe_dns_test.terraform_site.id
  ]
  
  db_tests = [
    terraprobe_db_test.postgres_test.id,
    terraprobe_db_test.mysql_test.id
  ]
}

output "http_test_results" {
  value = {
    passed      = terraprobe_http_test.github_status.test_passed
    status_code = terraprobe_http_test.github_status.last_status_code
    time_ms     = terraprobe_http_test.github_status.last_response_time
  }
}

output "tcp_test_results" {
  value = {
    passed        = terraprobe_tcp_test.google_dns.test_passed
    connect_time  = terraprobe_tcp_test.google_dns.last_connect_time
  }
}

output "dns_test_results" {
  value = {
    passed       = terraprobe_dns_test.terraform_site.test_passed
    result       = terraprobe_dns_test.terraform_site.last_result
    response_time = terraprobe_dns_test.terraform_site.last_result_time
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

output "test_suite_results" {
  value = {
    all_passed   = terraprobe_test_suite.all_tests.all_passed
    passed_count = terraprobe_test_suite.all_tests.passed_count
    total_count  = terraprobe_test_suite.all_tests.total_count
  }
} 