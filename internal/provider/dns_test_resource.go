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
var _ resource.Resource = &DnsTestResource{}
var _ resource.ResourceWithImportState = &DnsTestResource{}

func NewDnsTestResource() resource.Resource {
	return &DnsTestResource{}
}

// DnsTestResource defines the resource implementation.
type DnsTestResource struct {
	clientConfig *TerraProbeClientConfig
}

// DnsTestResourceModel describes the resource data model.
type DnsTestResourceModel struct {
	Name         types.String `tfsdk:"name"`
	Hostname     types.String `tfsdk:"hostname"`
	RecordType   types.String `tfsdk:"record_type"`
	ExpectResult types.String `tfsdk:"expect_result"`
	Resolver     types.String `tfsdk:"resolver"`
	Timeout      types.Int64  `tfsdk:"timeout"`
	Retries      types.Int64  `tfsdk:"retries"`
	RetryDelay   types.Int64  `tfsdk:"retry_delay"`
	Id           types.String `tfsdk:"id"`

	// Results
	LastRun        types.String `tfsdk:"last_run"`
	LastResult     types.String `tfsdk:"last_result"`
	LastResultTime types.Int64  `tfsdk:"last_result_time"`
	TestPassed     types.Bool   `tfsdk:"test_passed"`
	Error          types.String `tfsdk:"error"`
}

func (r *DnsTestResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_test"
}

func (r *DnsTestResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "DNS test resource that validates DNS resolution",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Descriptive name for the test",
				Required:            true,
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Hostname or domain to resolve",
				Required:            true,
			},
			"record_type": schema.StringAttribute{
				MarkdownDescription: "DNS record type to query (A, AAAA, CNAME, MX, TXT, etc.)",
				Required:            true,
			},
			"expect_result": schema.StringAttribute{
				MarkdownDescription: "Expected result in the DNS response (IP address, hostname, etc.)",
				Optional:            true,
			},
			"resolver": schema.StringAttribute{
				MarkdownDescription: "DNS resolver to use (e.g., 8.8.8.8, 1.1.1.1)",
				Optional:            true,
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Timeout in seconds for the DNS query",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0), // 0 means use provider default
			},
			"retries": schema.Int64Attribute{
				MarkdownDescription: "Number of retries for the DNS query",
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
			"last_result": schema.StringAttribute{
				MarkdownDescription: "Result from the last DNS query",
				Computed:            true,
			},
			"last_result_time": schema.Int64Attribute{
				MarkdownDescription: "Query time in milliseconds from the last test run",
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

func (r *DnsTestResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DnsTestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DnsTestResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Generate a unique identifier for this test
	data.Id = types.StringValue(fmt.Sprintf("dns-test-%s", time.Now().Format("20060102150405")))

	// Run the DNS test
	err := r.runTest(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("DNS Test Error", err.Error())
		return
	}

	// Set the last run time
	data.LastRun = types.StringValue(time.Now().Format(time.RFC3339))

	// Write logs
	tflog.Trace(ctx, "created DNS test resource")
	tflog.Debug(ctx, fmt.Sprintf("DNS Test Result: %t - %s", data.TestPassed.ValueBool(), data.Hostname.ValueString()))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DnsTestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DnsTestResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Run the DNS test to get the latest results
	err := r.runTest(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("DNS Test Error", err.Error())
		return
	}

	// Update the last run time
	data.LastRun = types.StringValue(time.Now().Format(time.RFC3339))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DnsTestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DnsTestResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Run the DNS test with the updated configuration
	err := r.runTest(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("DNS Test Error", err.Error())
		return
	}

	// Update the last run time
	data.LastRun = types.StringValue(time.Now().Format(time.RFC3339))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DnsTestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DnsTestResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Nothing special to do for delete, as this is a stateless resource
	// The resource will be removed from Terraform state
}

func (r *DnsTestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *DnsTestResource) runTest(ctx context.Context, data *DnsTestResourceModel) error {
	// Get timeout from resource or default from provider
	timeout := time.Second * 5
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

	// Set up DNS resolver
	resolver := net.DefaultResolver
	if !data.Resolver.IsNull() && data.Resolver.ValueString() != "" {
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: timeout,
				}
				return d.DialContext(ctx, "udp", data.Resolver.ValueString()+":53")
			},
		}
	}

	// Perform the DNS lookup with retries
	var result []string
	var lookupErr error
	var responseTime time.Duration

	recordType := data.RecordType.ValueString()
	for i := int64(0); i <= retries; i++ {
		start := time.Now()

		// Different lookup methods based on record type
		switch recordType {
		case "A", "AAAA":
			var ips []net.IP

			if recordType == "A" {
				// A record - return IPv4 addresses
				ips, lookupErr = resolver.LookupIP(ctx, "ip4", data.Hostname.ValueString())
			} else {
				// AAAA record - return IPv6 addresses
				ips, lookupErr = resolver.LookupIP(ctx, "ip6", data.Hostname.ValueString())
			}

			if lookupErr == nil {
				result = make([]string, len(ips))
				for i, ip := range ips {
					result[i] = ip.String()
				}
			}
		case "CNAME":
			var cname string
			cname, lookupErr = resolver.LookupCNAME(ctx, data.Hostname.ValueString())
			if lookupErr == nil {
				result = []string{cname}
			}
		case "MX":
			var mxs []*net.MX
			mxs, lookupErr = resolver.LookupMX(ctx, data.Hostname.ValueString())
			if lookupErr == nil {
				result = make([]string, len(mxs))
				for i, mx := range mxs {
					result[i] = fmt.Sprintf("%d %s", mx.Pref, mx.Host)
				}
			}
		case "TXT":
			result, lookupErr = resolver.LookupTXT(ctx, data.Hostname.ValueString())
		case "NS":
			var nss []*net.NS
			nss, lookupErr = resolver.LookupNS(ctx, data.Hostname.ValueString())
			if lookupErr == nil {
				result = make([]string, len(nss))
				for i, ns := range nss {
					result[i] = ns.Host
				}
			}
		default:
			lookupErr = fmt.Errorf("unsupported DNS record type: %s", recordType)
		}

		responseTime = time.Since(start)

		if lookupErr == nil {
			break
		}

		if i < retries {
			time.Sleep(retryDelay)
		}
	}

	// Handle DNS lookup errors
	if lookupErr != nil {
		data.Error = types.StringValue(fmt.Sprintf("DNS lookup failed: %s", lookupErr.Error()))
		data.TestPassed = types.BoolValue(false)
		data.LastResultTime = types.Int64Value(int64(responseTime / time.Millisecond))
		data.LastResult = types.StringValue("")
		return nil // Don't return error as we want to keep the error in the state
	}

	// Join the results into a comma-separated string
	resultStr := ""
	if len(result) > 0 {
		resultStr = result[0]
		for i := 1; i < len(result); i++ {
			resultStr += ", " + result[i]
		}
	}

	// Update the test results
	data.LastResultTime = types.Int64Value(int64(responseTime / time.Millisecond))
	data.LastResult = types.StringValue(resultStr)

	// Check if the test passed
	passed := true
	var errorMsg string

	// If an expected result is specified, check if it's in the actual results
	if !data.ExpectResult.IsNull() && data.ExpectResult.ValueString() != "" {
		expectResult := data.ExpectResult.ValueString()
		found := false

		for _, res := range result {
			if res == expectResult {
				found = true
				break
			}
		}

		if !found {
			passed = false
			errorMsg = fmt.Sprintf("Expected result '%s' not found in DNS response", expectResult)
		}
	}

	// Set the test result
	data.TestPassed = types.BoolValue(passed)

	// Set error message if test failed
	if !passed {
		data.Error = types.StringValue(errorMsg)
	} else {
		data.Error = types.StringValue("")
	}

	return nil
}
