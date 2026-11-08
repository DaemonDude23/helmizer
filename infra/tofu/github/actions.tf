# Any action may run, but only pinned to a full commit SHA (GitHub enforces
# it; Renovate keeps the pins current). The default GITHUB_TOKEN is read-only
# and workflows elevate per job.
resource "github_actions_repository_permissions" "this" {
  repository           = github_repository.this.name
  enabled              = true
  allowed_actions      = "all"
  sha_pinning_required = true
}

resource "github_workflow_repository_permissions" "this" {
  repository                       = github_repository.this.name
  default_workflow_permissions     = "read"
  can_approve_pull_request_reviews = false
}
