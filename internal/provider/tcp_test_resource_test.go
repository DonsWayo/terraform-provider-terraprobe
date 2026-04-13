package provider

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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

// TestTcpHardFailDiagnostic verifies that hard_fail only adds an error when
// the test did not pass.
func TestTcpHardFailDiagnostic(t *testing.T) {
	cases := []struct {
		name       string
		hardFail   bool
		testPassed bool
		wantErr    bool
	}{
		{"hard_fail off + failed", false, false, false},
		{"hard_fail off + passed", false, true, false},
		{"hard_fail on + passed", true, true, false},
		{"hard_fail on + failed", true, false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			data := &TcpTestResourceModel{
				Name:       types.StringValue("probe"),
				Host:       types.StringValue("127.0.0.1"),
				Port:       types.Int64Value(22),
				HardFail:   types.BoolValue(tc.hardFail),
				TestPassed: types.BoolValue(tc.testPassed),
				Error:      types.StringValue("connection refused"),
			}
			addTcpHardFailDiagnostic(&diags, data)
			if got := diags.HasError(); got != tc.wantErr {
				t.Fatalf("HasError()=%v, want %v (diags: %v)", got, tc.wantErr, diags.Errors())
			}
		})
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

// TestAccTcpTestResource_HardFail verifies that hard_fail=true propagates
// through terraform apply: passing tests succeed, failing tests raise an error.
func TestAccTcpTestResource_HardFail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping acceptance test in short mode")
	}

	// Bind a listener only so we know a port that is guaranteed free when closed.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate free port: %v", err)
	}
	host, portStr, _ := net.SplitHostPort(listener.Addr().String())
	_ = listener.Close()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"terraprobe": providerserver.NewProtocol6WithError(New("test")()),
		},
		Steps: []resource.TestStep{
			{
				Config: `
				provider "terraprobe" {}

				resource "terraprobe_tcp_test" "test" {
				  name      = "DNS Check"
				  host      = "8.8.8.8"
				  port      = 53
				  hard_fail = true
				}
				`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("terraprobe_tcp_test.test", "test_passed", "true"),
					resource.TestCheckResourceAttr("terraprobe_tcp_test.test", "hard_fail", "true"),
				),
			},
			{
				Config: fmt.Sprintf(`
				provider "terraprobe" {}

				resource "terraprobe_tcp_test" "test" {
				  name        = "Unreachable"
				  host        = %q
				  port        = %s
				  retries     = 0
				  timeout     = 1
				  hard_fail   = true
				}
				`, host, portStr),
				ExpectError: regexp.MustCompile(`hard_fail enabled`),
			},
		},
	})
}
