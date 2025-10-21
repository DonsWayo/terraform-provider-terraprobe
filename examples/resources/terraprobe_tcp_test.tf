resource "terraprobe_tcp_test" "database_connection" {
  name = "Database Connection Test"
  host = "db.example.com"
  port = 5432 # PostgreSQL port

  # Override provider defaults (optional)
  timeout     = 5
  retries     = 2
  retry_delay = 2
}

# Using interpolation with other resources
resource "terraprobe_tcp_test" "redis_connection" {
  name = "Redis Connection Test"
  host = aws_elasticache_cluster.redis.cache_nodes.0.address
  port = 6379 # Redis port

  # This ensures the test runs after the Redis cluster is created
  depends_on = [aws_elasticache_cluster.redis]
}

# Output test results
output "database_connection_test" {
  value = {
    passed          = terraprobe_tcp_test.database_connection.test_passed
    last_run        = terraprobe_tcp_test.database_connection.last_run
    connect_time_ms = terraprobe_tcp_test.database_connection.last_connect_time
    error           = terraprobe_tcp_test.database_connection.error
  }
} 