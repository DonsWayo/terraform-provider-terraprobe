resource "terraprobe_db_test" "postgres" {
  name     = "PostgreSQL Connection Test"
  type     = "postgres"
  host     = "db.example.com"
  port     = 5432
  username = "postgres"
  password = "your-password"
  database = "mydb"
  ssl_mode = "disable"
  query    = "SELECT 1"
}

resource "terraprobe_db_test" "mysql" {
  name     = "MySQL Connection Test"
  type     = "mysql"
  host     = "mysql.example.com"
  port     = 3306
  username = "root"
  password = "your-password"
  database = "mydb"
  query    = "SELECT 1"
}

output "db_test_results" {
  value = {
    passed        = terraprobe_db_test.postgres.test_passed
    query_time_ms = terraprobe_db_test.postgres.last_query_time
    rows_returned = terraprobe_db_test.postgres.last_result_rows
    error         = terraprobe_db_test.postgres.error
  }
}
