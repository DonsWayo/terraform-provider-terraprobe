// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TestSuiteResource{}
var _ resource.ResourceWithImportState = &TestSuiteResource{}

func NewTestSuiteResource() resource.Resource {
	return &TestSuiteResource{}
}

// TestSuiteResource defines the resource implementation.
type TestSuiteResource struct {
	clientConfig *TerraProbeClientConfig
}

// TestSuiteResourceModel describes the resource data model.
type TestSuiteResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	HttpTests   types.Set    `tfsdk:"http_tests"`
	TcpTests    types.Set    `tfsdk:"tcp_tests"`
	Id          types.String `tfsdk:"id"`

	// Results
	LastRun     types.String `tfsdk:"last_run"`
	AllPassed   types.Bool   `tfsdk:"all_passed"`
	PassedCount types.Int64  `tfsdk:"passed_count"`
	FailedCount types.Int64  `tfsdk:"failed_count"`
	TotalCount  types.Int64  `tfsdk:"total_count"`
	FailedTests types.List   `tfsdk:"failed_tests"`
}

func (r *TestSuiteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_test_suite"
}

func (r *TestSuiteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Test suite resource that groups multiple tests and provides aggregate results",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the test suite",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the test suite",
				Optional:            true,
			},
			"http_tests": schema.SetAttribute{
				MarkdownDescription: "List of HTTP test resource references",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"tcp_tests": schema.SetAttribute{
				MarkdownDescription: "List of TCP test resource references",
				Optional:            true,
				ElementType:         types.StringType,
			},

			// Results - these are computed values based on the last test run
			"last_run": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last test run",
				Computed:            true,
			},
			"all_passed": schema.BoolAttribute{
				MarkdownDescription: "Whether all tests passed",
				Computed:            true,
			},
			"passed_count": schema.Int64Attribute{
				MarkdownDescription: "Number of tests that passed",
				Computed:            true,
			},
			"failed_count": schema.Int64Attribute{
				MarkdownDescription: "Number of tests that failed",
				Computed:            true,
			},
			"total_count": schema.Int64Attribute{
				MarkdownDescription: "Total number of tests in the suite",
				Computed:            true,
			},
			"failed_tests": schema.ListAttribute{
				MarkdownDescription: "List of tests that failed",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Test suite identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *TestSuiteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	clientConfig, ok := req.ProviderData.(*TerraProbeClientConfig)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *TerraProbeClientConfig, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.clientConfig = clientConfig
}

func (r *TestSuiteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TestSuiteResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Generate a unique identifier for this test suite
	data.Id = types.StringValue(fmt.Sprintf("test-suite-%s", time.Now().Format("20060102150405")))

	// This is just a container of references to other resources.
	// The actual test results will be computed when the state is read.
	// For now, we'll just initialize the computed fields.
	data.LastRun = types.StringValue(time.Now().Format(time.RFC3339))

	// Calculate totals and initialize
	var httpTestsCount int64 = 0
	var tcpTestsCount int64 = 0

	if !data.HttpTests.IsNull() {
		httpTestsCount = int64(len(data.HttpTests.Elements()))
	}

	if !data.TcpTests.IsNull() {
		tcpTestsCount = int64(len(data.TcpTests.Elements()))
	}

	totalCount := httpTestsCount + tcpTestsCount
	data.TotalCount = types.Int64Value(totalCount)
	data.PassedCount = types.Int64Value(0)
	data.FailedCount = types.Int64Value(0)
	data.AllPassed = types.BoolValue(false)

	// Initialize an empty list of failed tests
	emptyList := []attr.Value{}
	data.FailedTests = types.ListValueMust(types.StringType, emptyList)

	// Write logs
	tflog.Trace(ctx, "created test suite resource")
	tflog.Debug(ctx, fmt.Sprintf("Test Suite Created: %s with %d tests", data.Name.ValueString(), totalCount))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TestSuiteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TestSuiteResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// In a real implementation, we would check the status of all referenced tests here
	// and calculate the aggregate results. For now, we'll just update the last run time.

	// Mock implementation to simulate test results calculation
	// In a real implementation, you would query the results of the referenced tests
	// from the Terraform state or make calls to evaluate them.

	// For simplicity in this initial version, we'll set some default values.
	// In a full implementation, you would:
	// 1. Get references to all test resources
	// 2. Read their test_passed attribute
	// 3. Calculate the aggregated results

	// Update the last run time
	data.LastRun = types.StringValue(time.Now().Format(time.RFC3339))

	// For now, just set some default values
	// In a real implementation, these would be calculated based on the referenced tests
	var httpTestsCount int64 = 0
	var tcpTestsCount int64 = 0

	if !data.HttpTests.IsNull() {
		httpTestsCount = int64(len(data.HttpTests.Elements()))
	}

	if !data.TcpTests.IsNull() {
		tcpTestsCount = int64(len(data.TcpTests.Elements()))
	}

	totalCount := httpTestsCount + tcpTestsCount
	data.TotalCount = types.Int64Value(totalCount)

	// In a real implementation, you would check each test's result
	// For this example, we'll assume all tests pass
	data.PassedCount = types.Int64Value(totalCount)
	data.FailedCount = types.Int64Value(0)
	data.AllPassed = types.BoolValue(true)

	// Empty list of failed tests
	emptyList := []attr.Value{}
	data.FailedTests = types.ListValueMust(types.StringType, emptyList)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TestSuiteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TestSuiteResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update the last run time
	data.LastRun = types.StringValue(time.Now().Format(time.RFC3339))

	// Calculate totals based on the updated test references
	var httpTestsCount int64 = 0
	var tcpTestsCount int64 = 0

	if !data.HttpTests.IsNull() {
		httpTestsCount = int64(len(data.HttpTests.Elements()))
	}

	if !data.TcpTests.IsNull() {
		tcpTestsCount = int64(len(data.TcpTests.Elements()))
	}

	totalCount := httpTestsCount + tcpTestsCount
	data.TotalCount = types.Int64Value(totalCount)

	// In a real implementation, you would check each test's result
	// For this example, we'll assume all tests pass
	data.PassedCount = types.Int64Value(totalCount)
	data.FailedCount = types.Int64Value(0)
	data.AllPassed = types.BoolValue(true)

	// Empty list of failed tests
	emptyList := []attr.Value{}
	data.FailedTests = types.ListValueMust(types.StringType, emptyList)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TestSuiteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TestSuiteResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Nothing special to do for delete, as this is a stateless resource
	// The resource will be removed from Terraform state
}

func (r *TestSuiteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
