package provider

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// TestDbTestResource_runTest tests the database test resource's runTest function
func TestDbTestResource_runTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This is a more involved test since it requires a database,
	// so we'll use Docker to spin up a test database.

	// Create a client config for testing
	clientConfig := &TerraProbeClientConfig{
		UserAgent:  "TerraProbe-Test",
		Retries:    1,
		RetryDelay: time.Second,
	}

	// Create the resource
	resource := &DbTestResource{
		clientConfig: clientConfig,
	}

	// Create a context for the test
	ctx := context.Background()

	// Basic mock test without actual DB connection
	t.Run("Mock test - unsupported database type", func(t *testing.T) {
		model := &DbTestResourceModel{
			Name:     types.StringValue("Invalid DB Test"),
			Type:     types.StringValue("invalid"),
			Host:     types.StringValue("localhost"),
			Port:     types.Int64Value(1234),
			Username: types.StringValue("user"),
			Password: types.StringValue("pass"),
			Database: types.StringValue("testdb"),
		}

		err := resource.runTest(ctx, model)
		if err == nil {
			t.Fatalf("Expected error for unsupported database type, but got none")
		}
	})

	// Set up a PostgreSQL container
	pgContainer, pgHost, pgPort, err := setupPostgres(t)
	if err != nil {
		t.Fatalf("Failed to set up PostgreSQL container: %v", err)
	}
	defer pgContainer.Close()

	// Test PostgreSQL connection
	t.Run("PostgreSQL connection test", func(t *testing.T) {
		model := &DbTestResourceModel{
			Name:     types.StringValue("PostgreSQL Test"),
			Type:     types.StringValue("postgres"),
			Host:     types.StringValue(pgHost),
			Port:     types.Int64Value(int64(pgPort)),
			Username: types.StringValue("postgres"),
			Password: types.StringValue("postgres"),
			Database: types.StringValue("postgres"),
			SSLMode:  types.StringValue("disable"),
			Query:    types.StringValue("SELECT 1"),
		}

		err := resource.runTest(ctx, model)
		if err != nil {
			t.Fatalf("PostgreSQL test failed: %v", err)
		}

		if !model.TestPassed.ValueBool() {
			t.Errorf("Expected PostgreSQL test to pass, but it failed with error: %s", model.Error.ValueString())
		}

		if model.LastResultRows.ValueInt64() != 1 {
			t.Errorf("Expected 1 row from PostgreSQL query, got %d", model.LastResultRows.ValueInt64())
		}
	})

	// Set up a MySQL container
	mysqlContainer, mysqlHost, mysqlPort, err := setupMySQL(t)
	if err != nil {
		t.Fatalf("Failed to set up MySQL container: %v", err)
	}
	defer mysqlContainer.Close()

	// Test MySQL connection
	t.Run("MySQL connection test", func(t *testing.T) {
		model := &DbTestResourceModel{
			Name:     types.StringValue("MySQL Test"),
			Type:     types.StringValue("mysql"),
			Host:     types.StringValue(mysqlHost),
			Port:     types.Int64Value(int64(mysqlPort)),
			Username: types.StringValue("root"),
			Password: types.StringValue("mysql"),
			Database: types.StringValue("mysql"),
			Query:    types.StringValue("SELECT 1"),
		}

		err := resource.runTest(ctx, model)
		if err != nil {
			t.Fatalf("MySQL test failed: %v", err)
		}

		if !model.TestPassed.ValueBool() {
			t.Errorf("Expected MySQL test to pass, but it failed with error: %s", model.Error.ValueString())
		}

		if model.LastResultRows.ValueInt64() != 1 {
			t.Errorf("Expected 1 row from MySQL query, got %d", model.LastResultRows.ValueInt64())
		}
	})

	// Test with invalid credentials
	t.Run("Invalid credentials test", func(t *testing.T) {
		model := &DbTestResourceModel{
			Name:     types.StringValue("Invalid Credentials Test"),
			Type:     types.StringValue("postgres"),
			Host:     types.StringValue(pgHost),
			Port:     types.Int64Value(int64(pgPort)),
			Username: types.StringValue("invaliduser"),
			Password: types.StringValue("invalidpass"),
			Database: types.StringValue("postgres"),
			SSLMode:  types.StringValue("disable"),
		}

		err := resource.runTest(ctx, model)
		if err != nil {
			t.Fatalf("Test with invalid credentials failed: %v", err)
		}

		if model.TestPassed.ValueBool() {
			t.Errorf("Expected test with invalid credentials to fail, but it passed")
		}
	})

	// Test with invalid query
	t.Run("Invalid query test", func(t *testing.T) {
		model := &DbTestResourceModel{
			Name:     types.StringValue("Invalid Query Test"),
			Type:     types.StringValue("postgres"),
			Host:     types.StringValue(pgHost),
			Port:     types.Int64Value(int64(pgPort)),
			Username: types.StringValue("postgres"),
			Password: types.StringValue("postgres"),
			Database: types.StringValue("postgres"),
			SSLMode:  types.StringValue("disable"),
			Query:    types.StringValue("SELECT * FROM nonexistenttable"),
		}

		err := resource.runTest(ctx, model)
		if err != nil {
			t.Fatalf("Test with invalid query failed: %v", err)
		}

		if model.TestPassed.ValueBool() {
			t.Errorf("Expected test with invalid query to fail, but it passed")
		}
	})
}

// TestAccDbTestResource is an acceptance test for the database test resource
func TestAccDbTestResource(t *testing.T) {
	// Skip in short mode as acceptance tests make real network connections
	if testing.Short() {
		t.Skip("skipping acceptance test in short mode")
	}

	// Set up Docker containers for the acceptance test
	pgContainer, err := setupDockerForAcceptanceTest(t)
	if err != nil {
		t.Skipf("Skipping acceptance test due to Docker setup failure: %v", err)
	}
	defer func() {
		if pgContainer != nil {
			pgContainer.Close()
		}
	}()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"terraprobe": providerserver.NewProtocol6WithError(New("test")()),
		},
		Steps: []resource.TestStep{
			{
				Config: `
				provider "terraprobe" {}
				
				resource "terraprobe_db_test" "local_postgres" {
				  name     = "Local PostgreSQL Test"
				  type     = "postgres"
				  host     = "localhost"
				  port     = 5432
				  username = "postgres"
				  password = "postgres"
				  database = "postgres"
				  ssl_mode = "disable"
				  query    = "SELECT 1"
				}
				`,
				// We'll check if we can connect to local PostgreSQL
				// If not, the test will be skipped
				SkipFunc: func() (bool, error) {
					// Try to connect to local PostgreSQL to see if it's available
					db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=postgres dbname=postgres sslmode=disable")
					if err != nil {
						return true, nil // Skip if we can't connect
					}

					err = db.Ping()
					db.Close()

					return err != nil, nil // Skip if ping fails
				},
			},
		},
	})
}

// Helper function to set up Docker container for acceptance test
func setupDockerForAcceptanceTest(t *testing.T) (*dockertest.Resource, error) {
	// Set up a PostgreSQL container for the acceptance test
	pgContainer, _, _, err := setupPostgres(t)
	if err != nil {
		return nil, err
	}
	return pgContainer, nil
}

// Helper functions to set up test databases using Docker
func setupPostgres(t *testing.T) (*dockertest.Resource, string, int, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, "", 0, err
	}

	// Create a PostgreSQL container
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14",
		Env: []string{
			"POSTGRES_USER=postgres",
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_DB=postgres",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		return nil, "", 0, err
	}

	// Determine host and port
	host := "localhost"
	port := resource.GetPort("5432/tcp")
	portInt := 0

	// Convert port string to int
	_, err = fmt.Sscanf(port, "%d", &portInt)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to parse port: %v", err)
	}

	// Wait for PostgreSQL to be ready
	if err := pool.Retry(func() error {
		db, err := sql.Open("postgres",
			fmt.Sprintf("host=%s port=%s user=postgres password=postgres dbname=postgres sslmode=disable",
				host, port))
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		return nil, "", 0, err
	}

	return resource, host, portInt, nil
}

func setupMySQL(t *testing.T) (*dockertest.Resource, string, int, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, "", 0, err
	}

	// Create a MySQL container
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mysql",
		Tag:        "8.0",
		Env: []string{
			"MYSQL_ROOT_PASSWORD=mysql",
			"MYSQL_DATABASE=mysql",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		return nil, "", 0, err
	}

	// Determine host and port
	host := "localhost"
	port := resource.GetPort("3306/tcp")
	portInt := 0

	// Convert port string to int
	_, err = fmt.Sscanf(port, "%d", &portInt)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to parse port: %v", err)
	}

	// Wait for MySQL to be ready
	if err := pool.Retry(func() error {
		db, err := sql.Open("mysql",
			fmt.Sprintf("root:mysql@tcp(%s:%s)/mysql", host, port))
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		return nil, "", 0, err
	}

	return resource, host, portInt, nil
}
