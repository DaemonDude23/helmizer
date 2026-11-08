# infra/tofu/github

Manages the GitHub repository **DaemonDude23/helmizer** with the
[`integrations/github`](https://search.opentofu.org/provider/integrations/github/latest)
provider.

| Resource | Notes |
| --- | --- |
| `github_repository` | Visibility, merge methods (squash/rebase only), cleanup, secret scanning, Pages |
| `github_repository_ruleset.default_branch` | No deletion or force-push, linear history, PRs need `ci-ok`; admins bypass |
| `github_repository_ruleset.release_tags` | `v*` tags cannot be moved or deleted |
| `github_repository_dependabot_security_updates` | Off: Renovate opens security updates |
| `github_actions_repository_permissions` / `github_workflow_repository_permissions` | Read-only default token |

## Run

```sh
make tofu-plan     # GITHUB_TOKEN from `gh auth token`; read-only
make tofu-apply    # applies the saved planfile
```

## New repository vs. adopting an existing one

- **New**: leave `adopt = false`. The first apply creates the repository, and you push to it.
- **Existing**: set `adopt = true` in `terragrunt.hcl` inputs. If the repository already has
  rulesets, add their ids to `ruleset_ids`
  (`gh api repos/DaemonDude23/helmizer/rulesets --jq '.[] | "\(.id) \(.name)"'`).
  Plan until the diff shows only intended changes.

## Not managed here

- Installing the **Mend Renovate** GitHub app on the repository. For personal accounts the
  app's repository selection is UI-only: <https://github.com/apps/renovate>.
- Classic branch protection rules. Rulesets replace them, so delete old ones by hand after
  the rulesets apply.
