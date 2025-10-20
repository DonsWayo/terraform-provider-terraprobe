package provider

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TcpTestResource{}
var _ resource.ResourceWithImportState = &TcpTestResource{}

func NewTcpTestResource() resource.Resource {
	return &TcpTestResource{}
}

// TcpTestResource defines the resource implementation.
type TcpTestResource struct {
	clientConfig *TerraProbeClientConfig
}

// TcpTestResourceModel describes the resource data model.
type TcpTestResourceModel struct {
	Name       types.String `tfsdk:"name"`
	Host       types.String `tfsdk:"host"`
	Port       types.Int64  `tfsdk:"port"`
	Timeout    types.Int64  `tfsdk:"timeout"`
	Retries    types.Int64  `tfsdk:"retries"`
	RetryDelay types.Int64  `tfsdk:"retry_delay"`
	Id         types.String `tfsdk:"id"`

	// Results
	LastRun         types.String `tfsdk:"last_run"`
	LastConnectTime types.Int64  `tfsdk:"last_connect_time"`
	TestPassed      types.Bool   `tfsdk:"test_passed"`
	Error           types.String `tfsdk:"error"`
}

func (r *TcpTestResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tcp_test"
}

func (r *TcpTestResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "TCP test resource that validates TCP connectivity to a host and port",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Descriptive name for the test",
				Required:            true,
			},
			"host": schema.StringAttribute{
				MarkdownDescription: "Host to connect to (IP address or hostname)",
				Required:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "Port to connect to",
				Required:            true,
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Timeout in seconds for the connection attempt",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0), // 0 means use provider default
			},
			"retries": schema.Int64Attribute{
				MarkdownDescription: "Number of retries for the connection attempt",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0), // 0 means use provider default
			},
			"retry_delay": schema.Int64Attribute{
				MarkdownDescription: "Delay between retries in seconds",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0), // 0 means use provider default
			},

			// Results - these are computed values based on the last test run
			"last_run": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last test run",
				Computed:            true,
			},
			"last_connect_time": schema.Int64Attribute{
				MarkdownDescription: "Connection time in milliseconds from the last test run",
				Computed:            true,
			},
			"test_passed": schema.BoolAttribute{
				MarkdownDescription: "Whether the test passed (connection was established)",
				Computed:            true,
			},
			"error": schema.StringAttribute{
				MarkdownDescription: "Error message if the test failed",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Test identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *TcpTestResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TcpTestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TcpTestResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Generate a unique identifier for this test
	data.Id = types.StringValue(fmt.Sprintf("tcp-test-%s", time.Now().Format("20060102150405")))

	// Run the TCP test
	err := r.runTest(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("TCP Test Error", err.Error())
		return
	}

	// Set the last run time
	data.LastRun = types.StringValue(time.Now().Format(time.RFC3339))

	// Write logs
	tflog.Trace(ctx, "created TCP test resource")
	tflog.Debug(ctx, fmt.Sprintf("TCP Test Result: %t - %s:%d", data.TestPassed.ValueBool(), data.Host.ValueString(), data.Port.ValueInt64()))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TcpTestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TcpTestResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Run the TCP test again during Read
	err := r.runTest(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("TCP Test Error", err.Error())
		return
	}

	// Update the last run time
	data.LastRun = types.StringValue(time.Now().Format(time.RFC3339))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TcpTestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TcpTestResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Run the TCP test with updated parameters
	err := r.runTest(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("TCP Test Error", err.Error())
		return
	}

	// Update the last run time
	data.LastRun = types.StringValue(time.Now().Format(time.RFC3339))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TcpTestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TcpTestResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Nothing special to do for delete, as this is a stateless resource
	// The resource will be removed from Terraform state
}

func (r *TcpTestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// runTest runs the TCP test and updates the resource model with the results.
func (r *TcpTestResource) runTest(_ context.Context, data *TcpTestResourceModel) error {
	// Get timeout from resource or default from provider
	timeout := r.clientConfig.HttpClient.Timeout
	if !data.Timeout.IsNull() && data.Timeout.ValueInt64() > 0 {
		timeout = time.Duration(data.Timeout.ValueInt64()) * time.Second
	}

	// Get retries from resource or default from provider
	retries := r.clientConfig.Retries
	if !data.Retries.IsNull() && data.Retries.ValueInt64() > 0 {
		retries = data.Retries.ValueInt64()
	}

	// Get retry delay from resource or default from provider
	retryDelay := r.clientConfig.RetryDelay
	if !data.RetryDelay.IsNull() && data.RetryDelay.ValueInt64() > 0 {
		retryDelay = time.Duration(data.RetryDelay.ValueInt64()) * time.Second
	}

	// Format the address
	address := fmt.Sprintf("%s:%d", data.Host.ValueString(), data.Port.ValueInt64())

	// Perform the connection attempt with retries
	var err error
	var connectTime time.Duration

	for i := int64(0); i <= retries; i++ {
		start := time.Now()
		// Try to establish a TCP connection
		conn, dialErr := net.DialTimeout("tcp", address, timeout)
		connectTime = time.Since(start)

		if dialErr == nil {
			// Connection successful
			_ = conn.Close()
			err = nil
			break
		}

		err = dialErr

		if i < retries {
			time.Sleep(retryDelay)
		}
	}

	// Handle connection errors
	if err != nil {
		data.Error = types.StringValue(fmt.Sprintf("TCP connection failed: %s", err.Error()))
		data.TestPassed = types.BoolValue(false)
		data.LastConnectTime = types.Int64Value(0)
		return nil // Don't return error as we want to keep the error in the state
	}

	// Update the test results
	data.LastConnectTime = types.Int64Value(int64(connectTime / time.Millisecond))
	data.TestPassed = types.BoolValue(true)
	data.Error = types.StringValue("")

	return nil
}
