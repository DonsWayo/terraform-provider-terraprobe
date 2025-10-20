package provider

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// Database drivers.
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

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
var _ resource.Resource = &DbTestResource{}
var _ resource.ResourceWithImportState = &DbTestResource{}

func NewDbTestResource() resource.Resource {
	return &DbTestResource{}
}

// DbTestResource defines the resource implementation.
type DbTestResource struct {
	clientConfig *TerraProbeClientConfig
}

// DbTestResourceModel describes the resource data model.
type DbTestResourceModel struct {
	Name       types.String `tfsdk:"name"`
	Type       types.String `tfsdk:"type"`
	Host       types.String `tfsdk:"host"`
	Port       types.Int64  `tfsdk:"port"`
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	Database   types.String `tfsdk:"database"`
	Query      types.String `tfsdk:"query"`
	Timeout    types.Int64  `tfsdk:"timeout"`
	Retries    types.Int64  `tfsdk:"retries"`
	RetryDelay types.Int64  `tfsdk:"retry_delay"`
	Id         types.String `tfsdk:"id"`

	// Additional connection options
	SSLMode     types.String `tfsdk:"ssl_mode"`
	MaxLifetime types.Int64  `tfsdk:"max_lifetime"`
	MaxIdleConn types.Int64  `tfsdk:"max_idle_conn"`
	MaxOpenConn types.Int64  `tfsdk:"max_open_conn"`

	// Results
	LastRun        types.String `tfsdk:"last_run"`
	LastQueryTime  types.Int64  `tfsdk:"last_query_time"`
	LastResultRows types.Int64  `tfsdk:"last_result_rows"`
	TestPassed     types.Bool   `tfsdk:"test_passed"`
	Error          types.String `tfsdk:"error"`
}

func (r *DbTestResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_db_test"
}

func (r *DbTestResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Database test resource that validates database connectivity and queries",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Descriptive name for the test",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type of database (mysql, postgres)",
				Required:            true,
			},
			"host": schema.StringAttribute{
				MarkdownDescription: "Database host to connect to",
				Required:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "Database port",
				Required:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Database username",
				Required:            true,
				Sensitive:           true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Database password",
				Required:            true,
				Sensitive:           true,
			},
			"database": schema.StringAttribute{
				MarkdownDescription: "Database name to connect to",
				Required:            true,
			},
			"query": schema.StringAttribute{
				MarkdownDescription: "SQL query to execute (default: SELECT 1)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("SELECT 1"),
			},
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Timeout in seconds for the database connection and query",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0), // 0 means use provider default
			},
			"retries": schema.Int64Attribute{
				MarkdownDescription: "Number of retries for the database connection",
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
			"ssl_mode": schema.StringAttribute{
				MarkdownDescription: "SSL mode for the database connection (disable, require, verify-ca, verify-full)",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("disable"),
			},
			"max_lifetime": schema.Int64Attribute{
				MarkdownDescription: "Maximum lifetime of a connection in seconds",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
			},
			"max_idle_conn": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of idle connections",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(2),
			},
			"max_open_conn": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of open connections",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(5),
			},

			// Results - these are computed values based on the last test run
			"last_run": schema.StringAttribute{
				MarkdownDescription: "Timestamp of the last test run",
				Computed:            true,
			},
			"last_query_time": schema.Int64Attribute{
				MarkdownDescription: "Query time in milliseconds from the last test run",
				Computed:            true,
			},
			"last_result_rows": schema.Int64Attribute{
				MarkdownDescription: "Number of rows returned by the query",
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

func (r *DbTestResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DbTestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DbTestResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Generate a unique identifier for this test
	data.Id = types.StringValue(fmt.Sprintf("db-test-%s", time.Now().Format("20060102150405")))

	// Run the database test
	err := r.runTest(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Database Test Error", err.Error())
		return
	}

	// Set the last run time
	data.LastRun = types.StringValue(time.Now().Format(time.RFC3339))

	// Write logs
	tflog.Trace(ctx, "created database test resource")
	tflog.Debug(ctx, fmt.Sprintf("Database Test Result: %t - %s:%d/%s", data.TestPassed.ValueBool(), data.Host.ValueString(), data.Port.ValueInt64(), data.Database.ValueString()))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DbTestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DbTestResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Run the database test to get the latest results
	err := r.runTest(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Database Test Error", err.Error())
		return
	}

	// Update the last run time
	data.LastRun = types.StringValue(time.Now().Format(time.RFC3339))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DbTestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DbTestResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Run the database test with the updated configuration
	err := r.runTest(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Database Test Error", err.Error())
		return
	}

	// Update the last run time
	data.LastRun = types.StringValue(time.Now().Format(time.RFC3339))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DbTestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DbTestResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Nothing special to do for delete, as this is a stateless resource
	// The resource will be removed from Terraform state
}

func (r *DbTestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// runTest performs the database test.
func (r *DbTestResource) runTest(ctx context.Context, data *DbTestResourceModel) error {
	// Get timeout from resource or default from provider
	timeout := time.Second * 10
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

	// Create a database connection string based on the database type
	var connStr string
	dbType := data.Type.ValueString()

	switch dbType {
	case "mysql":
		connStr = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			data.Username.ValueString(),
			data.Password.ValueString(),
			data.Host.ValueString(),
			data.Port.ValueInt64(),
			data.Database.ValueString())
	case "postgres":
		sslMode := data.SSLMode.ValueString()
		connStr = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			data.Host.ValueString(),
			data.Port.ValueInt64(),
			data.Username.ValueString(),
			data.Password.ValueString(),
			data.Database.ValueString(),
			sslMode)
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}

	// Create a context with timeout for the database operations
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Perform the database connection and query with retries
	var db *sql.DB
	var openErr error
	var rowCount int64
	var queryTime time.Duration

	for i := int64(0); i <= retries; i++ {
		// Open the database connection
		db, openErr = sql.Open(dbType, connStr)
		if openErr != nil {
			if i < retries {
				time.Sleep(retryDelay)
				continue
			}
			break
		}

		// Configure the connection pool
		if !data.MaxLifetime.IsNull() && data.MaxLifetime.ValueInt64() > 0 {
			db.SetConnMaxLifetime(time.Duration(data.MaxLifetime.ValueInt64()) * time.Second)
		}
		if !data.MaxIdleConn.IsNull() {
			db.SetMaxIdleConns(int(data.MaxIdleConn.ValueInt64()))
		}
		if !data.MaxOpenConn.IsNull() {
			db.SetMaxOpenConns(int(data.MaxOpenConn.ValueInt64()))
		}

		// Ping the database to check the connection
		pingErr := db.PingContext(timeoutCtx)
		if pingErr != nil {
			db.Close()
			if i < retries {
				time.Sleep(retryDelay)
				continue
			}
			openErr = pingErr
			break
		}

		// Execute the query
		start := time.Now()
		query := data.Query.ValueString()
		rows, queryErr := db.QueryContext(timeoutCtx, query)
		queryTime = time.Since(start)

		if queryErr != nil {
			db.Close()
			if i < retries {
				time.Sleep(retryDelay)
				continue
			}
			openErr = queryErr
			break
		}

		// Count the rows
		rowCount = 0
		for rows.Next() {
			rowCount++
		}

		// Check for errors during row iteration
		rowErr := rows.Err()
		rows.Close()

		if rowErr != nil {
			db.Close()
			if i < retries {
				time.Sleep(retryDelay)
				continue
			}
			openErr = rowErr
			break
		}

		// Close the database connection
		db.Close()
		openErr = nil
		break
	}

	// Handle errors
	if openErr != nil {
		data.Error = types.StringValue(fmt.Sprintf("Database test failed: %s", openErr.Error()))
		data.TestPassed = types.BoolValue(false)
		data.LastQueryTime = types.Int64Value(0)
		data.LastResultRows = types.Int64Value(0)
		return nil // Don't return error as we want to keep the error in the state
	}

	// Update the results
	data.LastQueryTime = types.Int64Value(int64(queryTime / time.Millisecond))
	data.LastResultRows = types.Int64Value(rowCount)
	data.TestPassed = types.BoolValue(true)
	data.Error = types.StringValue("")

	return nil
}
