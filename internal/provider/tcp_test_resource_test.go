package provider

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestTcpTestResource_runTest tests the TCP test resource's runTest function.
func TestTcpTestResource_runTest(t *testing.T) {
	// Set up a TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to set up TCP listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	// Start a goroutine to handle connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	// Create client config
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}
	clientConfig := &TerraProbeClientConfig{
		HttpClient: httpClient,
		UserAgent:  "TerraProbe-Test",
		Retries:    1,
		RetryDelay: time.Second,
	}

	// Create the resource
	resource := &TcpTestResource{
		clientConfig: clientConfig,
	}

	// Parse host and port from listener address
	host, portStr, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.ParseInt(portStr, 10, 64)

	// Create the resource model
	model := &TcpTestResourceModel{
		Name: types.StringValue("Test TCP"),
		Host: types.StringValue(host),
		Port: types.Int64Value(port),
	}

	// Run the test
	ctx := context.Background()
	err = resource.runTest(ctx, model)
	if err != nil {
		t.Fatalf("runTest failed: %v", err)
	}

	// Check the results
	if !model.TestPassed.ValueBool() {
		t.Errorf("Expected test to pass, but it failed with error: %s", model.Error.ValueString())
	}

	// Test with failing condition - wrong port
	model.Port = types.Int64Value(1) // Use a port that's unlikely to be listening
	err = resource.runTest(ctx, model)
	if err != nil {
		t.Fatalf("runTest failed: %v", err)
	}

	if model.TestPassed.ValueBool() {
		t.Errorf("Expected test to fail with wrong port, but it passed")
	}
}

// TestAccTcpTestResource is an acceptance test for the TCP test resource.
func TestAccTcpTestResource(t *testing.T) {
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
				
				resource "terraprobe_tcp_test" "test" {
				  name = "DNS Check"
				  host = "8.8.8.8"
				  port = 53
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("terraprobe_tcp_test.test", "test_passed", "true"),
				),
			},
		},
	})
}
