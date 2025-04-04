# Define some individual tests
resource "terraprobe_http_test" "api_health" {
  name  = "API Health Check"
  url   = "https://api.example.com/health"
  
  expect_status_code = 200
  expect_contains   = "\"status\":\"healthy\""
}

resource "terraprobe_tcp_test" "database_connection" {
  name = "Database Connection Test"
  host = "db.example.com"
  port = 5432
}

# Group tests in a test suite
resource "terraprobe_test_suite" "production_validation" {
  name        = "Production Health Checks"
  description = "Validates all production services are healthy"
  
  # Reference the above tests
  http_tests = [
    terraprobe_http_test.api_health.id
  ]
  
  tcp_tests = [
    terraprobe_tcp_test.database_connection.id
  ]
}

# Output comprehensive test results
output "test_suite_results" {
  value = {
    name         = terraprobe_test_suite.production_validation.name
    last_run     = terraprobe_test_suite.production_validation.last_run
    all_passed   = terraprobe_test_suite.production_validation.all_passed
    passed_count = terraprobe_test_suite.production_validation.passed_count
    failed_count = terraprobe_test_suite.production_validation.failed_count
    total_count  = terraprobe_test_suite.production_validation.total_count
    failed_tests = terraprobe_test_suite.production_validation.failed_tests
  }
} 