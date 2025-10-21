## 0.2.0 (Unreleased)

IMPROVEMENTS:

* Updated Go dependencies to latest compatible versions
* Go 1.25.3 compatibility
* terraform-plugin-framework 1.16.1
* terraform-plugin-go 0.29.0
* terraform-plugin-testing 1.13.3
* docker-cli 28.5.1, docker 28.5.1
* terraform-plugin-sdk/v2 2.38.1
* Enhanced security and stability updates across dependencies

## 0.1.0 (2025-10-21)

FEATURES:

* Initial release of the TerraProbe provider
* Provider configuration for default timeout, retries, and retry delay
* `terraprobe_http_test` resource for validating HTTP endpoints
* `terraprobe_tcp_test` resource for validating TCP connectivity
* `terraprobe_dns_test` resource for validating DNS resolution
* `terraprobe_db_test` resource for validating database connectivity
* `terraprobe_test_suite` resource for grouping tests with aggregate results
* Detailed test results including response time, status code, and content validation
* Support for Terraform 1.13.* and OpenTofu 1.10.*
