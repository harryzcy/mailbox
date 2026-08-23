terraform {
  # Partial config: bucket is supplied at init time via -backend-config so the
  # name stays out of this public repo. See README for setup.
  backend "s3" {
    key          = "mailbox/dev/terraform.tfstate"
    region       = "us-west-2"
    encrypt      = true
    use_lockfile = true
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.61.0"
    }
  }

  required_version = ">= 1.14"
}
