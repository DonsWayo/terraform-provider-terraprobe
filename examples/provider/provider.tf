terraform {
  required_providers {
    terraprobe = {
      source = "hashicorp/terraprobe"
    }
  }
  required_version = ">= 1.1.0"
}

provider "terraprobe" {
  default_timeout     = 30
  default_retries     = 3
  default_retry_delay = 5
  user_agent          = "TerraProbe Example"
}
