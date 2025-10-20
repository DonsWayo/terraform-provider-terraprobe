// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestProviderConfigure tests the provider's configuration
func TestProviderConfigure(t *testing.T) {
	// Create a provider instance
	p := New("test")()

	// Verify the provider type is correct
	if p == nil {
		t.Fatal("provider is nil")
	}

	// Make sure creating the provider doesn't panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("provider panicked: %v", r)
			}
		}()
		_ = New("test")()
	}()
}

// TestAccProvider is an acceptance test for the provider
func TestAccProvider(t *testing.T) {
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
				provider "terraprobe" {
				  retries = 2
				  retry_delay = 2
				}
				`,
			},
		},
	})
}
