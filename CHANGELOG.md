## 0.3.0 (2026-04-13)

FEATURES:

* Added `hard_fail` attribute to every test resource (`terraprobe_http_test`, `terraprobe_tcp_test`, `terraprobe_dns_test`, `terraprobe_db_test`, and `terraprobe_test_suite`). When set to `true`, the resource returns an error after exhausting all retries without passing, so `terraform apply` fails. Defaults to `false`, preserving prior behaviour. ([#27](https://github.com/DonsWayo/terraform-provider-terraprobe/issues/27), [#35](https://github.com/DonsWayo/terraform-provider-terraprobe/pull/35))

## 0.2.1 (2025-10-21)

BUG FIXES:

* Fixed incorrect provider source in documentation (was `hashicorp/terraprobe`, now correctly `DonsWayo/terraprobe`)
* Fixed incorrect provider source in examples
* Removed obsolete scaffolding examples
* Added provider description to terraform-registry-manifest.json for better registry display

## 0.2.0 (2025-10-21)

IMPROVEMENTS:

* Updated Go dependencies to latest compatible versions
* Go 1.25.3 compatibility
* terraform-plugin-framework 1.16.1
* terraform-plugin-go 0.29.0
* terraform-plugin-testing 1.13.3
* docker-cli 28.5.1, docker 28.5.1
* terraform-plugin-sdk/v2 2.38.1
* Enhanced security and stability updates across dependencies
* GPG signing enabled for all releases
* Repository renamed to terraform-provider-terraprobe for Terraform Registry compliance

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
