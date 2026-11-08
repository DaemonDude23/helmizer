# infra/tofu

OpenTofu + Terragrunt for this project's own footprint: one **unit per provider**,
each with its **own local state**, in the style of lab-k8s.

```text
infra/tofu/
  root.hcl       # shared: state pinned next to each unit, planfile plan/apply, optional tfvars
  .gitignore     # state, tfvars, planfiles, caches: never committed
  .tflint.hcl
  github/        # the github.com project: settings, protection, github-specific automation
```

## Running

```sh
make tofu-plan              # UNIT=github by default; read-only, writes ./planfile
make tofu-apply             # applies exactly that planfile
```

The Makefile exports the API token from the forge CLI (`gh auth token` /
`glab config get token`), so no token is stored anywhere.

## State

State is **local and gitignored**. It lives in the owner's checkout only. A clone
without it can re-adopt every existing resource: set `adopt = true` in the unit's
`terragrunt.hcl` inputs and plan. The import blocks pick the live objects up,
and the plan should show no changes.
