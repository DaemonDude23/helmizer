terraform {
  required_version = ">= 1.10.0"

  required_providers {
    # https://search.opentofu.org/provider/integrations/github/latest
    github = {
      source  = "integrations/github"
      version = "6.13.0"
    }
  }
}

# Authenticates with GITHUB_TOKEN from the environment.
provider "github" {
  owner = var.owner
}
