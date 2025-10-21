resource "terraprobe_dns_test" "example_a_record" {
  name        = "Example.com A Record Test"
  hostname    = "example.com"
  record_type = "A"
}

resource "terraprobe_dns_test" "example_ipv6" {
  name        = "Example.com IPv6 Test"
  hostname    = "example.com"
  record_type = "AAAA"
}

resource "terraprobe_dns_test" "mail_servers" {
  name        = "Email Server MX Records"
  hostname    = "example.com"
  record_type = "MX"
}

output "dns_test_results" {
  value = {
    passed         = terraprobe_dns_test.example_a_record.test_passed
    result_time_ms = terraprobe_dns_test.example_a_record.last_result_time
    dns_records    = terraprobe_dns_test.example_a_record.last_result
    error          = terraprobe_dns_test.example_a_record.error
  }
}
