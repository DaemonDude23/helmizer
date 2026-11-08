# Adoption of an existing repository (adopt = true): every resource that
# already exists on GitHub is imported, so the first plan shows only the
# settings this config changes. Imports are no-ops once the state has them.
locals {
  adopt = var.adopt ? toset(["this"]) : toset([])
}

import {
  for_each = local.adopt
  to       = github_repository.this
  id       = var.repository
}

import {
  for_each = local.adopt
  to       = github_repository_vulnerability_alerts.this
  id       = var.repository
}

import {
  for_each = local.adopt
  to       = github_branch_default.this
  id       = var.repository
}

import {
  for_each = local.adopt
  to       = github_repository_dependabot_security_updates.this
  id       = var.repository
}

import {
  for_each = local.adopt
  to       = github_actions_repository_permissions.this
  id       = var.repository
}

import {
  for_each = local.adopt
  to       = github_workflow_repository_permissions.this
  id       = var.repository
}

import {
  for_each = local.rulesets && contains(keys(var.ruleset_ids), "default_branch") ? toset(["this"]) : toset([])
  to       = github_repository_ruleset.default_branch[0]
  id       = "${var.repository}:${var.ruleset_ids["default_branch"]}"
}

import {
  for_each = local.rulesets && contains(keys(var.ruleset_ids), "release_tags") ? toset(["this"]) : toset([])
  to       = github_repository_ruleset.release_tags[0]
  id       = "${var.repository}:${var.ruleset_ids["release_tags"]}"
}

import {
  for_each = var.adopt && var.pages && var.pages_exists ? toset(["this"]) : toset([])
  to       = github_repository_pages.this[0]
  id       = var.repository
}
