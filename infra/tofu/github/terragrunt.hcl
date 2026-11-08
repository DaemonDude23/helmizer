# The GitHub repository this project lives in: settings, rulesets, Actions
# policy. State (local, gitignored) and plan/apply handling come from root.hcl.
#
# The provider reads GITHUB_TOKEN; `make tofu-plan` sets it from `gh auth token`.
include "root" {
  path = find_in_parent_folders("root.hcl")
}

inputs = {
  owner          = "DaemonDude23"
  repository     = "helmizer"
  description    = "Generates kustomization.yaml for your locally-rendered YAML manifests, such as with Helm templates"
  visibility     = "public"
  default_branch = "main"
  topics         = ["kubernetes", "helm", "yaml", "glob", "wildcard", "kustomization", "kustomize"]
  # The repository exists: import it and its "main" ruleset.
  adopt       = true
  ruleset_ids = { default_branch = 24824172 }
  # Dependency review was already required before the footprint migration.
  required_checks = ["ci-ok", "dependency review"]

  pages        = true
  homepage_url = "https://daemondude23.github.io/helmizer/"
}
