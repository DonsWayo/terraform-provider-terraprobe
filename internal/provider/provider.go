package provider

import (
	"context"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure TerraProbeProvider satisfies various provider interfaces.
var _ provider.Provider = &TerraProbeProvider{}
var _ provider.ProviderWithFunctions = &TerraProbeProvider{}
var _ provider.ProviderWithEphemeralResources = &TerraProbeProvider{}

// TerraProbeProvider defines the provider implementation.
type TerraProbeProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// TerraProbeProviderModel describes the provider data model.
type TerraProbeProviderModel struct {
	DefaultTimeout    types.Int64  `tfsdk:"default_timeout"`
	DefaultRetries    types.Int64  `tfsdk:"default_retries"`
	DefaultRetryDelay types.Int64  `tfsdk:"default_retry_delay"`
	UserAgent         types.String `tfsdk:"user_agent"`
}

func (p *TerraProbeProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "terraprobe"
	resp.Version = p.version
}

func (p *TerraProbeProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The TerraProbe provider allows you to validate infrastructure after deployment through various tests. This provider is designed to integrate with your regular Terraform workflow.",

		Attributes: map[string]schema.Attribute{
			"default_timeout": schema.Int64Attribute{
				MarkdownDescription: "Default timeout in seconds for all tests. Can be overridden at the resource level.",
				Optional:            true,
			},
			"default_retries": schema.Int64Attribute{
				MarkdownDescription: "Default number of retries for all tests. Can be overridden at the resource level.",
				Optional:            true,
			},
			"default_retry_delay": schema.Int64Attribute{
				MarkdownDescription: "Default delay between retries in seconds. Can be overridden at the resource level.",
				Optional:            true,
			},
			"user_agent": schema.StringAttribute{
				MarkdownDescription: "User agent to use for HTTP requests.",
				Optional:            true,
			},
		},
	}
}

func (p *TerraProbeProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config TerraProbeProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Set default values if not provided
	timeout := 30 * time.Second
	if !config.DefaultTimeout.IsNull() {
		timeout = time.Duration(config.DefaultTimeout.ValueInt64()) * time.Second
	}

	retries := int64(3)
	if !config.DefaultRetries.IsNull() {
		retries = config.DefaultRetries.ValueInt64()
	}

	retryDelay := 5 * time.Second
	if !config.DefaultRetryDelay.IsNull() {
		retryDelay = time.Duration(config.DefaultRetryDelay.ValueInt64()) * time.Second
	}

	userAgent := "TerraProbe Terraform Provider"
	if !config.UserAgent.IsNull() {
		userAgent = config.UserAgent.ValueString()
	}

	// Create a custom HTTP client with the specified timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create a client configuration
	clientConfig := &TerraProbeClientConfig{
		HttpClient: client,
		UserAgent:  userAgent,
		Retries:    retries,
		RetryDelay: retryDelay,
	}

	resp.DataSourceData = clientConfig
	resp.ResourceData = clientConfig
}

// TerraProbeClientConfig contains the provider-level configuration for client operations
type TerraProbeClientConfig struct {
	HttpClient *http.Client
	UserAgent  string
	Retries    int64
	RetryDelay time.Duration
}

func (p *TerraProbeProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewHttpTestResource,
		NewTcpTestResource,
		NewTestSuiteResource,
	}
}

func (p *TerraProbeProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		// In the future, we may add ephemeral resources for one-time tests
	}
}

func (p *TerraProbeProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// We'll implement the test results data source later
	}
}

func (p *TerraProbeProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		// We may add utility functions in the future
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TerraProbeProvider{
			version: version,
		}
	}
}
