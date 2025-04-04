package provider

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestDnsTestResource_runTest tests the DNS test resource's runTest function
func TestDnsTestResource_runTest(t *testing.T) {
	// Create a client config for testing
	clientConfig := &TerraProbeClientConfig{
		UserAgent:  "TerraProbe-Test",
		Retries:    1,
		RetryDelay: time.Second,
	}

	// Create the resource
	resource := &DnsTestResource{
		clientConfig: clientConfig,
	}

	// Create a context for the test
	ctx := context.Background()

	// Test Case 1: A record for a well-known domain
	t.Run("A Record for google.com", func(t *testing.T) {
		model := &DnsTestResourceModel{
			Name:       types.StringValue("Google DNS Test"),
			Hostname:   types.StringValue("google.com"),
			RecordType: types.StringValue("A"),
		}

		err := resource.runTest(ctx, model)
		if err != nil {
			t.Fatalf("runTest failed: %v", err)
		}

		// The test should pass since google.com should have an A record
		if !model.TestPassed.ValueBool() {
			t.Errorf("Expected test to pass, but it failed with error: %s", model.Error.ValueString())
		}

		// There should be some result
		if model.LastResult.ValueString() == "" {
			t.Errorf("Expected non-empty result for google.com A record")
		}
	})

	// Test Case 2: MX record for a well-known email domain
	t.Run("MX Record for gmail.com", func(t *testing.T) {
		model := &DnsTestResourceModel{
			Name:       types.StringValue("Gmail MX Test"),
			Hostname:   types.StringValue("gmail.com"),
			RecordType: types.StringValue("MX"),
		}

		err := resource.runTest(ctx, model)
		if err != nil {
			t.Fatalf("runTest failed: %v", err)
		}

		// The test should pass since gmail.com should have MX records
		if !model.TestPassed.ValueBool() {
			t.Errorf("Expected test to pass, but it failed with error: %s", model.Error.ValueString())
		}

		// There should be some result
		if model.LastResult.ValueString() == "" {
			t.Errorf("Expected non-empty result for gmail.com MX record")
		}
	})

	// Test Case 3: Test with expected result that doesn't match
	t.Run("Test with wrong expected result", func(t *testing.T) {
		model := &DnsTestResourceModel{
			Name:         types.StringValue("Wrong Expectation Test"),
			Hostname:     types.StringValue("google.com"),
			RecordType:   types.StringValue("A"),
			ExpectResult: types.StringValue("999.999.999.999"), // This IP shouldn't exist in the results
		}

		err := resource.runTest(ctx, model)
		if err != nil {
			t.Fatalf("runTest failed: %v", err)
		}

		// The test should fail because the expected result doesn't match
		if model.TestPassed.ValueBool() {
			t.Errorf("Expected test to fail with wrong expectation, but it passed")
		}
	})

	// Test Case 4: Non-existent domain
	t.Run("Non-existent domain", func(t *testing.T) {
		model := &DnsTestResourceModel{
			Name:       types.StringValue("Non-existent Domain Test"),
			Hostname:   types.StringValue("thisdomain.should.not.exist.example"),
			RecordType: types.StringValue("A"),
		}

		err := resource.runTest(ctx, model)
		if err != nil {
			t.Fatalf("runTest failed: %v", err)
		}

		// The test should fail because the domain doesn't exist
		if model.TestPassed.ValueBool() {
			t.Errorf("Expected test to fail for non-existent domain, but it passed")
		}
	})

	// Test Case 5: Unsupported record type
	t.Run("Unsupported record type", func(t *testing.T) {
		model := &DnsTestResourceModel{
			Name:       types.StringValue("Unsupported Record Type Test"),
			Hostname:   types.StringValue("google.com"),
			RecordType: types.StringValue("INVALID"),
		}

		err := resource.runTest(ctx, model)
		if err != nil {
			t.Fatalf("runTest failed: %v", err)
		}

		// The test should fail because the record type is not supported
		if model.TestPassed.ValueBool() {
			t.Errorf("Expected test to fail for unsupported record type, but it passed")
		}
	})
}

// TestAccDnsTestResource is an acceptance test for the DNS test resource
func TestAccDnsTestResource(t *testing.T) {
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
				
				resource "terraprobe_dns_test" "test" {
				  name        = "DNS A Record Test"
				  hostname    = "www.terraform.io"
				  record_type = "A"
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("terraprobe_dns_test.test", "test_passed", "true"),
					resource.TestCheckResourceAttrSet("terraprobe_dns_test.test", "last_result"),
				),
			},
		},
	})
}
