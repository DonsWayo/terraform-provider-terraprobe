package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestHttpTestResource_runTest tests the HTTP test resource's runTest function
func TestHttpTestResource_runTest(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()

	// Create a client config for testing
	clientConfig := &TerraProbeClientConfig{
		HttpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		UserAgent:  "TerraProbe-Test",
		Retries:    1,
		RetryDelay: time.Second,
	}

	// Create the resource
	resource := &HttpTestResource{
		clientConfig: clientConfig,
	}

	// Create the resource model
	model := &HttpTestResourceModel{
		Name:             types.StringValue("Test HTTP"),
		URL:              types.StringValue(server.URL),
		Method:           types.StringValue("GET"),
		ExpectStatusCode: types.Int64Value(200),
		ExpectContains:   types.StringValue("status"),
	}

	// Create a context for the test
	ctx := context.Background()

	// Run the test
	err := resource.runTest(ctx, model)
	if err != nil {
		t.Fatalf("runTest failed: %v", err)
	}

	// Check the results
	if !model.TestPassed.ValueBool() {
		t.Errorf("Expected test to pass, but it failed with error: %s", model.Error.ValueString())
	}

	if model.LastStatusCode.ValueInt64() != 200 {
		t.Errorf("Expected status code 200, got %d", model.LastStatusCode.ValueInt64())
	}

	// Test with failing condition - wrong status code expectation
	model.ExpectStatusCode = types.Int64Value(404)
	err = resource.runTest(ctx, model)
	if err != nil {
		t.Fatalf("runTest failed: %v", err)
	}

	if model.TestPassed.ValueBool() {
		t.Errorf("Expected test to fail with status code 404, but it passed")
	}
}

// TestAccHttpTestResource is an acceptance test for the HTTP test resource
func TestAccHttpTestResource(t *testing.T) {
	// Skip in short mode as acceptance tests make real API calls
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
				
				resource "terraprobe_http_test" "test" {
				  name  = "HTTP Test"
				  url   = "https://www.githubstatus.com/"
				  
				  expect_status_code = 200
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("terraprobe_http_test.test", "test_passed", "true"),
					resource.TestCheckResourceAttr("terraprobe_http_test.test", "last_status_code", "200"),
				),
			},
		},
	})
}
