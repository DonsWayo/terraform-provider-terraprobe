resource "terraprobe_http_test" "example_api" {
  name  = "Example API Health Check"
  url   = "https://api.example.com/health"
  
  # Optional configuration
  method = "GET"
  headers = {
    "Accept" = "application/json"
  }
  
  # Test expectations
  expect_status_code = 200
  expect_contains   = "\"status\":\"healthy\""
  
  # Override provider defaults (optional)
  timeout     = 10
  retries     = 2
  retry_delay = 3
}

# Using interpolation with other resources
resource "terraprobe_http_test" "load_balancer_check" {
  name = "Load Balancer Health Check"
  url  = "http://${aws_lb.example.dns_name}/health"
  
  # Define custom expectations
  expect_status_code = 200
  expect_contains   = "OK"
  
  # This ensures the test runs after the load balancer is created
  depends_on = [aws_lb.example]
}

# Output test results
output "api_test_results" {
  value = {
    passed           = terraprobe_http_test.example_api.test_passed
    last_run         = terraprobe_http_test.example_api.last_run
    status_code      = terraprobe_http_test.example_api.last_status_code
    response_time_ms = terraprobe_http_test.example_api.last_response_time
    error            = terraprobe_http_test.example_api.error
  }
} 