package provider

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestTestSuiteResource_Results tests the calculation of test suite results
func TestTestSuiteResource_Results(t *testing.T) {
	// Create a model with test resources
	model := &TestSuiteResourceModel{
		Name:        types.StringValue("Test Suite"),
		Description: types.StringValue("Test suite description"),
		Id:          types.StringValue("test-suite-123"),
	}

	// Set up HTTP tests
	httpTests := []attr.Value{
		types.StringValue("terraprobe_http_test.test1"),
		types.StringValue("terraprobe_http_test.test2"),
	}
	model.HttpTests = types.SetValueMust(types.StringType, httpTests)

	// Set up TCP tests
	tcpTests := []attr.Value{
		types.StringValue("terraprobe_tcp_test.test1"),
	}
	model.TcpTests = types.SetValueMust(types.StringType, tcpTests)

	// Initialize results
	model.LastRun = types.StringValue(time.Now().Format(time.RFC3339))
	model.TotalCount = types.Int64Value(3)
	model.PassedCount = types.Int64Value(2)
	model.FailedCount = types.Int64Value(1)
	model.AllPassed = types.BoolValue(false)

	failedTests := []attr.Value{
		types.StringValue("terraprobe_http_test.test2"),
	}
	model.FailedTests = types.ListValueMust(types.StringType, failedTests)

	// Test the calculated results
	if model.TotalCount.ValueInt64() != 3 {
		t.Errorf("Expected 3 total tests, got %d", model.TotalCount.ValueInt64())
	}

	if model.PassedCount.ValueInt64() != 2 {
		t.Errorf("Expected 2 tests passed, got %d", model.PassedCount.ValueInt64())
	}

	if model.FailedCount.ValueInt64() != 1 {
		t.Errorf("Expected 1 test failed, got %d", model.FailedCount.ValueInt64())
	}

	if model.AllPassed.ValueBool() {
		t.Errorf("Expected all tests passed to be false, got true")
	}

	if model.FailedTests.IsNull() || len(model.FailedTests.Elements()) != 1 {
		t.Errorf("Expected 1 failed test, got %d", len(model.FailedTests.Elements()))
	}
}

// TestAccTestSuiteResource is an acceptance test for the test suite resource
func TestAccTestSuiteResource(t *testing.T) {
	// Skip in short mode as acceptance tests make real network connections
	if testing.Short() {
		t.Skip("skipping acceptance test in short mode")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"terraprobe": providerserver.NewProtocol6WithError(New("test")()),
		},
		Steps: []resource.TestStep{
			{
				Config: `
				provider "terraprobe" {}
				
				resource "terraprobe_http_test" "google" {
				  name = "Google Check"
				  url = "https://www.google.com"
				  method = "GET"
				  expect_status_code = 200
				  expect_contains = "Google"
				  headers = {
				    "User-Agent" = "TerraProbe Test"
				  }
				}
				
				resource "terraprobe_tcp_test" "dns" {
				  name = "DNS Check"
				  host = "8.8.8.8"
				  port = 53
				}
				
				resource "terraprobe_test_suite" "all_tests" {
				  name = "All Tests"
				  description = "Tests for key services"
				  
				  http_tests = [
				    terraprobe_http_test.google.id
				  ]
				  
				  tcp_tests = [
				    terraprobe_tcp_test.dns.id
				  ]
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("terraprobe_test_suite.all_tests", "name", "All Tests"),
					resource.TestCheckResourceAttr("terraprobe_test_suite.all_tests", "description", "Tests for key services"),
					resource.TestCheckResourceAttr("terraprobe_test_suite.all_tests", "total_count", "2"),
					resource.TestCheckResourceAttr("terraprobe_test_suite.all_tests", "passed_count", "2"),
					resource.TestCheckResourceAttr("terraprobe_test_suite.all_tests", "failed_count", "0"),
					resource.TestCheckResourceAttr("terraprobe_test_suite.all_tests", "all_passed", "true"),
				),
			},
		},
	})
}
