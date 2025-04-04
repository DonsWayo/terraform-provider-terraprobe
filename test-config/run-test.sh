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

# Change to the root directory
cd ..

# Run unit tests if requested
if [[ "$MODE" == "unit-test" || "$MODE" == "all" ]]; then
  echo "Running Go unit tests..."
  go test ./internal/provider/... -v
  if [ $? -ne 0 ]; then
    echo "Unit tests failed"
    exit 1
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
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/hashicorp/terraprobe/0.1.0/darwin_amd64/
cp terraform-provider-terraprobe ~/.terraform.d/plugins/registry.terraform.io/hashicorp/terraprobe/0.1.0/darwin_amd64/

# Return to test directory
cd test-config

# Set Terraform configuration directory
export TF_CLI_CONFIG_FILE=$(pwd)/.terraformrc

# If we're in test or all mode, run Terraform
if [[ "$MODE" == "apply" || "$MODE" == "test" || "$MODE" == "all" ]]; then
  echo "Running Terraform apply..."
  terraform init
  terraform apply -auto-approve
  
  # Show output
  echo "Terraform apply completed. Output:"
  terraform output
fi

echo "All tests completed successfully" 