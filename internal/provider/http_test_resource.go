package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &HttpTestResource{}
var _ resource.ResourceWithImportState = &HttpTestResource{}

func NewHttpTestResource() resource.Resource {
	return &HttpTestResource{}
}

// HttpTestResource defines the resource implementation.
type HttpTestResource struct {
	clientConfig *TerraProbeClientConfig
}

// HttpTestResourceModel describes the resource data model.
type HttpTestResourceModel struct {
	Name             types.String `tfsdk:"name"`
	URL              types.String `tfsdk:"url"`
	Method           types.String `tfsdk:"method"`
	Headers          types.Map    `tfsdk:"headers"`
	Body             types.String `tfsdk:"body"`
	Timeout          types.Int64  `tfsdk:"timeout"`
	Retries          types.Int64  `tfsdk:"retries"`
	RetryDelay       types.Int64  `tfsdk:"retry_delay"`
	ExpectStatusCode types.Int64  `tfsdk:"expect_status_code"`
	ExpectContains   types.String `tfsdk:"expect_contains"`
	Id               types.String `tfsdk:"id"`

	// Results
	LastRun          types.String `tfsdk:"last_run"`
	LastStatusCode   types.Int64  `tfsdk:"last_status_code"`
	LastResponseBody types.String `tfsdk:"last_response_body"`
	LastResponseTime types.Int64  `tfsdk:"last_response_time"`
	TestPassed       types.Bool   `tfsdk:"test_passed"`
	Error            types.String `tfsdk:"error"`
}

func (r *HttpTestResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_http_test"
}

func (r *HttpTestResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "HTTP test resource that validates a HTTP endpoint",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Descriptive name for the test",
				Required:            true,
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "URL to test",
				Required:            true,
			},
			"method": schema.StringAttribute{
				MarkdownDescription: "HTTP method to use (GET, POST, PUT, DELETE, etc.)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("GET"),
			},
			"headers": schema.MapAttribute{
				MarkdownDescription: "HTTP headers to include in the request",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"body": schema.StringAttribute{
				MarkdownDescription: "Request body for POST, PUT, etc.",
				Optional:            true,
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Timeout in seconds for the HTTP request",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0), // 0 means use provider default
			},
			"retries": schema.Int64Attribute{
				MarkdownDescription: "Number of retries for the HTTP request",
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
			"expect_status_code": schema.Int64Attribute{
				MarkdownDescription: "Expected HTTP status code",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(200),
			},
			"expect_contains": schema.StringAttribute{
				MarkdownDescription: "String to look for in the response body",
				Optional:            true,
			},

			// Results - these are computed values based on the last test run
			"last_run": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last test run",
				Computed:            true,
			},
			"last_status_code": schema.Int64Attribute{
				MarkdownDescription: "Status code from the last test run",
				Computed:            true,
			},
			"last_response_body": schema.StringAttribute{
				MarkdownDescription: "Response body from the last test run",
				Computed:            true,
			},
			"last_response_time": schema.Int64Attribute{
				MarkdownDescription: "Response time in milliseconds from the last test run",
				Computed:            true,
			},
			"test_passed": schema.BoolAttribute{
				MarkdownDescription: "Whether the test passed",
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

func (r *HttpTestResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *HttpTestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data HttpTestResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Generate a unique identifier for this test
	data.Id = types.StringValue(fmt.Sprintf("http-test-%s", time.Now().Format("20060102150405")))

	// Run the HTTP test
	err := r.runTest(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("HTTP Test Error", err.Error())
		return
	}

	// Set the last run time
	data.LastRun = types.StringValue(time.Now().Format(time.RFC3339))

	// Write logs
	tflog.Trace(ctx, "created HTTP test resource")
	tflog.Debug(ctx, fmt.Sprintf("HTTP Test Result: %t - %s", data.TestPassed.ValueBool(), data.URL.ValueString()))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HttpTestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data HttpTestResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Run the HTTP test again during Read
	err := r.runTest(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("HTTP Test Error", err.Error())
		return
	}

	// Update the last run time
	data.LastRun = types.StringValue(time.Now().Format(time.RFC3339))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HttpTestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data HttpTestResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Run the HTTP test with updated parameters
	err := r.runTest(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("HTTP Test Error", err.Error())
		return
	}

	// Update the last run time
	data.LastRun = types.StringValue(time.Now().Format(time.RFC3339))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *HttpTestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data HttpTestResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Nothing special to do for delete, as this is a stateless resource
	// The resource will be removed from Terraform state
}

func (r *HttpTestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// runTest runs the HTTP test and updates the resource model with the results.
func (r *HttpTestResource) runTest(ctx context.Context, data *HttpTestResourceModel) error {
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

	// Create a custom client with the specified timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create the request
	method := "GET"
	if !data.Method.IsNull() {
		method = data.Method.ValueString()
	}

	var body io.Reader
	if !data.Body.IsNull() {
		body = strings.NewReader(data.Body.ValueString())
	}

	req, err := http.NewRequestWithContext(ctx, method, data.URL.ValueString(), body)
	if err != nil {
		data.Error = types.StringValue(fmt.Sprintf("Failed to create request: %s", err.Error()))
		data.TestPassed = types.BoolValue(false)
		return nil // Don't return error as we want to keep the error in the state
	}

	// Add headers
	if !data.Headers.IsNull() {
		headers := make(map[string]string)
		data.Headers.ElementsAs(ctx, &headers, false)

		for k, v := range headers {
			req.Header.Add(k, v)
		}
	}

	// Add user agent
	req.Header.Set("User-Agent", r.clientConfig.UserAgent)

	// Perform the request with retries
	var resp *http.Response
	var respErr error
	var responseTime time.Duration

	for i := int64(0); i <= retries; i++ {
		start := time.Now()
		resp, respErr = client.Do(req)
		responseTime = time.Since(start)

		if respErr == nil {
			break
		}

		if i < retries {
			time.Sleep(retryDelay)
		}
	}

	// Handle request errors
	if respErr != nil {
		data.Error = types.StringValue(fmt.Sprintf("Request failed: %s", respErr.Error()))
		data.TestPassed = types.BoolValue(false)
		data.LastResponseTime = types.Int64Value(0)
		data.LastStatusCode = types.Int64Value(0)
		data.LastResponseBody = types.StringValue("")
		return nil // Don't return error as we want to keep the error in the state
	}
	defer resp.Body.Close()

	// Read the response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		data.Error = types.StringValue(fmt.Sprintf("Failed to read response body: %s", err.Error()))
		data.TestPassed = types.BoolValue(false)
		data.LastResponseTime = types.Int64Value(int64(responseTime / time.Millisecond))
		data.LastStatusCode = types.Int64Value(int64(resp.StatusCode))
		data.LastResponseBody = types.StringValue("")
		return nil // Don't return error as we want to keep the error in the state
	}

	// Update the test results
	data.LastResponseTime = types.Int64Value(int64(responseTime / time.Millisecond))
	data.LastStatusCode = types.Int64Value(int64(resp.StatusCode))
	data.LastResponseBody = types.StringValue(string(respBody))

	// Check if the test passed
	passed := true
	var errorMsg strings.Builder

	// Check status code if expected is specified
	expectedStatusCode := int64(200)
	if !data.ExpectStatusCode.IsNull() {
		expectedStatusCode = data.ExpectStatusCode.ValueInt64()
	}

	if int64(resp.StatusCode) != expectedStatusCode {
		passed = false
		errorMsg.WriteString(fmt.Sprintf("Expected status code %d but got %d. ", expectedStatusCode, resp.StatusCode))
	}

	// Check response body contains expected string if specified
	if !data.ExpectContains.IsNull() && data.ExpectContains.ValueString() != "" {
		if !strings.Contains(string(respBody), data.ExpectContains.ValueString()) {
			passed = false
			errorMsg.WriteString(fmt.Sprintf("Response body does not contain '%s'. ", data.ExpectContains.ValueString()))
		}
	}

	// Set the test result
	data.TestPassed = types.BoolValue(passed)

	// Set error message if test failed
	if !passed {
		data.Error = types.StringValue(errorMsg.String())
	} else {
		data.Error = types.StringValue("")
	}

	return nil
}
