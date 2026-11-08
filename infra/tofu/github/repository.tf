locals {
  public   = var.visibility == "public"
  rulesets = coalesce(var.rulesets, local.public)
}

resource "github_repository" "this" {
  name         = var.repository
  description  = var.description
  homepage_url = var.homepage_url
  visibility   = var.visibility
  topics       = var.topics
  is_template  = var.is_template

  has_issues      = true
  has_discussions = false
  has_projects    = false
  has_wiki        = false

  # Squash or rebase only, linear history, branches cleaned up after merge.
  allow_merge_commit          = false
  allow_squash_merge          = true
  allow_rebase_merge          = true
  allow_auto_merge            = true
  allow_update_branch         = true
  delete_branch_on_merge      = true
  squash_merge_commit_title   = "PR_TITLE"
  squash_merge_commit_message = "PR_BODY"

  archive_on_destroy = true

  dynamic "security_and_analysis" {
    # Free for public repositories; private ones need GitHub Advanced Security.
    for_each = local.public ? [1] : []
    content {
      secret_scanning {
        status = "enabled"
      }
      secret_scanning_push_protection {
        status = "enabled"
      }
    }
  }

  lifecycle {
    prevent_destroy = true
  }
}

# GitHub Pages, deployed by the Pages workflow (no publishing branch).
resource "github_repository_pages" "this" {
  count      = var.pages ? 1 : 0
  repository = github_repository.this.name
  build_type = "workflow"
}

resource "github_repository_vulnerability_alerts" "this" {
  repository = github_repository.this.name
  enabled    = true
}

resource "github_branch_default" "this" {
  repository = github_repository.this.name
  branch     = var.default_branch
}

# Renovate (vulnerabilityAlerts) opens the security updates; Dependabot's
# would duplicate them.
resource "github_repository_dependabot_security_updates" "this" {
  repository = github_repository.this.name
  enabled    = false
}
