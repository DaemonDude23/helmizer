# Shared terragrunt configuration for every unit under infra/tofu/ (lab-k8s style).
#
# Units opt in with:
#
#   include "root" {
#     path = find_in_parent_folders("root.hcl")
#   }
#
# Terragrunt v1 requires the explicit filename.

# Pin state next to each unit.
#
# Without this block terragrunt keeps local state inside the ephemeral
# .terragrunt-cache working dir, where it is silently lost. State is local and
# NEVER committed (see .gitignore here): it lives in the owner's checkout and
# is carried by the sync that backs that checkout up. A fresh clone has no
# state; recover it by copying terraform.tfstate into the unit dir, or adopt
# the live resources again with the unit's import blocks (adopt = true).
remote_state {
  backend = "local"

  generate = {
    path      = "backend.tf"
    if_exists = "overwrite"
  }

  config = {
    path = "${get_terragrunt_dir()}/terraform.tfstate"
  }
}

terraform {
  # Plan to a file and apply exactly that file, so an apply can never pick up
  # drift that appeared between the two commands.
  extra_arguments "plan" {
    commands  = ["plan"]
    arguments = ["-out", "./planfile"]
  }

  extra_arguments "apply" {
    commands  = ["apply"]
    arguments = ["./planfile"]
  }

  # Secret inputs live in a gitignored terraform.tfvars next to the unit
  # (copy terraform.tfvars.example). Optional, so units without secrets work.
  extra_arguments "tfvars" {
    commands           = get_terraform_commands_that_need_vars()
    optional_var_files = ["${get_terragrunt_dir()}/terraform.tfvars"]
  }
}
