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

# Test suite combining both tests
resource "terraprobe_test_suite" "basic_connectivity" {
  name        = "Basic Connectivity Tests"
  description = "Tests basic internet connectivity"
  
  http_tests = [
    terraprobe_http_test.github_status.id
  ]
  
  tcp_tests = [
    terraprobe_tcp_test.google_dns.id
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

output "test_suite_results" {
  value = {
    all_passed   = terraprobe_test_suite.basic_connectivity.all_passed
    passed_count = terraprobe_test_suite.basic_connectivity.passed_count
    total_count  = terraprobe_test_suite.basic_connectivity.total_count
  }
} 