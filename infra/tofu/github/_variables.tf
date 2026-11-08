variable "owner" {
  type        = string
  description = "GitHub user or organization that owns the repository."
}

variable "repository" {
  type        = string
  description = "Repository name."
}

variable "description" {
  type        = string
  description = "Repository description (shown on GitHub)."
}

variable "visibility" {
  type        = string
  default     = "public"
  description = "public or private."

  validation {
    condition     = contains(["public", "private"], var.visibility)
    error_message = "visibility must be public or private."
  }
}

variable "topics" {
  type        = list(string)
  default     = []
  description = "Repository topics."
}

variable "homepage_url" {
  type        = string
  default     = ""
  description = "Homepage URL, usually the docs site. Empty for none."
}

variable "default_branch" {
  type        = string
  default     = "main"
  description = "Default branch, which the default-branch ruleset protects."
}

variable "is_template" {
  type        = bool
  default     = false
  description = "Mark the repository as a template (the footprint blueprints are)."
}

variable "pages" {
  type        = bool
  default     = false
  description = "Enable GitHub Pages, deployed by a workflow (build_type = workflow)."
}

variable "required_checks" {
  type        = list(string)
  default     = ["ci-ok"]
  description = "Status checks the default branch requires. ci-ok aggregates the CI workflow."
}

variable "rulesets" {
  type        = bool
  default     = null
  description = <<-EOT
    Manage the branch and tag rulesets. Defaults to true for public
    repositories. Private repositories of free personal accounts cannot
    use rulesets, so they default to false.
  EOT
}

variable "pages_exists" {
  type        = bool
  default     = false
  description = "When adopting: Pages is already enabled on the repository, so import it."
}

variable "adopt" {
  type        = bool
  default     = false
  description = "The repository already exists: import it (imports.tf) instead of creating it."
}

variable "ruleset_ids" {
  type        = map(number)
  default     = {}
  description = <<-EOT
    Existing ruleset ids to import when adopting, keyed default_branch and
    release_tags. List them with:
    gh api repos/OWNER/REPO/rulesets --jq '.[] | "\(.id) \(.name)"'
  EOT
}
