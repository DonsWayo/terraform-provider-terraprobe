#!/bin/bash

# Default mode
MODE="apply"

# Parse arguments
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    --test)
      MODE="test"
      shift
      ;;
    --unit)
      MODE="unit-test"
      shift
      ;;
    --all)
      MODE="all"
      shift
      ;;
    *)
      echo "Unknown option: $key"
      echo "Usage: $0 [--test|--unit|--all]"
      echo "  --test: Run Terraform apply to test the provider"
      echo "  --unit: Run Go unit tests"
      echo "  --all: Run both tests and Terraform apply"
      exit 1
      ;;
  esac
done

# Check if Docker is available
if ! command -v docker &> /dev/null; then
  echo "Docker is not installed or not available in PATH. Some database tests may fail."
  DOCKER_AVAILABLE=false
else
  # Check if Docker daemon is running
  if ! docker info &> /dev/null; then
    echo "Docker daemon is not running. Some database tests may fail."
    DOCKER_AVAILABLE=false
  else
    echo "Docker is available. Database tests will use Docker containers."
    DOCKER_AVAILABLE=true
  fi
fi

# Change to the root directory
cd ..

# Run unit tests if requested
if [[ "$MODE" == "unit-test" || "$MODE" == "all" ]]; then
  echo "Running Go unit tests..."
  
  # Run the tests
  go test ./internal/provider/... -v
  TEST_EXIT_CODE=$?
  
  if [ $TEST_EXIT_CODE -ne 0 ]; then
    echo "Unit tests failed with exit code $TEST_EXIT_CODE"
    exit $TEST_EXIT_CODE
  fi
  
  echo "Unit tests completed successfully"
  
  # Exit if only unit tests were requested
  if [[ "$MODE" == "unit-test" ]]; then
    exit 0
  fi
fi

# Build the provider
echo "Building provider..."
go build -o terraform-provider-terraprobe
BUILD_EXIT_CODE=$?
if [ $BUILD_EXIT_CODE -ne 0 ]; then
  echo "Build failed with exit code $BUILD_EXIT_CODE"
  exit $BUILD_EXIT_CODE
fi

# Create directories for the terraform plugin if they don't exist
PLUGIN_DIR=~/.terraform.d/plugins/registry.terraform.io/hashicorp/terraprobe/0.1.0/darwin_amd64/
mkdir -p $PLUGIN_DIR

# Copy the provider to the plugin directory
cp terraform-provider-terraprobe $PLUGIN_DIR
COPY_EXIT_CODE=$?
if [ $COPY_EXIT_CODE -ne 0 ]; then
  echo "Failed to copy provider to plugin directory: $COPY_EXIT_CODE"
  exit $COPY_EXIT_CODE
fi

# Return to test directory
cd test-config

# Set Terraform configuration directory
export TF_CLI_CONFIG_FILE=$(pwd)/.terraformrc

# If we're in test or all mode, run Terraform
if [[ "$MODE" == "apply" || "$MODE" == "test" || "$MODE" == "all" ]]; then
  echo "Running Terraform apply..."
  
  # Force recreation of the .terraform directory
  rm -rf .terraform
  
  # Run terraform init and apply
  terraform init
  INIT_EXIT_CODE=$?
  if [ $INIT_EXIT_CODE -ne 0 ]; then
    echo "Terraform init failed with exit code $INIT_EXIT_CODE"
    exit $INIT_EXIT_CODE
  fi
  
  terraform apply -auto-approve
  APPLY_EXIT_CODE=$?
  if [ $APPLY_EXIT_CODE -ne 0 ]; then
    echo "Terraform apply failed with exit code $APPLY_EXIT_CODE"
    exit $APPLY_EXIT_CODE
  fi
  
  # Show output
  echo "Terraform apply completed. Output:"
  terraform output
fi

echo "All tests completed successfully" 