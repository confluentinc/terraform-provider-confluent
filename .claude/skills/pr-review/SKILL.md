---
name: pr-review
description:
  Reviews pull requests for the Confluent Terraform provider. Use when reviewing PRs, doing
  self-review before sharing with the team, or when the user mentions "review PR", "help with PR",
  "review changes", "self-review", "review local changes", or "check my PR". Focuses on SDKv2 schema
  correctness, resource registration, deprecation/backward-compat, state upgraders, and the
  acceptance/WireMock/live test tiers.
allowed-tools:
  - Read
  - Bash
  - Grep
  - Glob
  - Task
---

# PR Review Skill

Reviews pull requests for the Confluent Terraform provider (Go + terraform-plugin-sdk/v2), focusing
on the failure modes that corrupt user state or break backward compatibility and that no linter here
will catch (this repo has no `golangci-lint` — only `make checkfmt`).

The review-relevant conventions live in `.claude/rules/resource-schema.md` and
`.claude/rules/acceptance-tests.md`, which auto-load when you edit matching files. This skill applies
them to a diff; consult the rules and existing sibling resources rather than re-deriving conventions.

## Two Review Modes

The mode is selected by the invocation context, not by the user. If the user supplies a PR
number/URL or asks about someone else's PR, run **Formal Review Mode**. Otherwise (no PR number,
working from a local branch, phrases like "self-review" or "check my PR") run **Self-Review Mode**.
When ambiguous, ask which mode to use.

### Self-Review Mode (for PR authors)

Use when: the author wants to check their own changes before sharing with the team.

Goals:

- Catch issues early, before formal review
- Verify a new resource/data source is registered and its schema is internally consistent
- Confirm stored-shape changes ship a `SchemaVersion` bump + state upgrader
- Confirm acceptance-test coverage (WireMock-backed where possible)

### Formal Review Mode (for reviewers)

Use when: a reviewer needs to evaluate a PR from another team member.

Goals:

- Quickly understand the scope and purpose of changes
- Identify state-safety and backward-compatibility risks
- Provide constructive feedback grounded in SDKv2 + Confluent conventions
- Verify the PR template checklist is honored

## Review Process

### Step 1: Gather Information

**For local changes (self-review):**

```bash
git diff main --name-only
git diff main --stat
git diff main
```

**For GitHub PRs:**

```bash
gh pr view <PR_NUMBER> --json number,title,body,author,baseRefName,headRefName,additions,deletions,changedFiles,state,reviewDecision
gh pr view --json number,title,body,author,baseRefName,headRefName,additions,deletions,changedFiles,state,reviewDecision
gh pr diff <PR_NUMBER>
gh pr view <PR_NUMBER> --json reviews,comments
gh issue view <ISSUE_NUMBER> --json body,comments
```

### Step 2: Filter Files for Review

**SKIP or skim** (generated or infrastructure):

- `vendor/**`
- `.semaphore/**`, `.goreleaser*.yml`, `.golicense.hcl` — release/CI

**DO review carefully** (small file, big blast radius):

- `internal/provider/provider.go` (`ResourcesMap` / `DataSourcesMap` registration)
- `internal/provider/constants.go` (shared deprecation messages)
- `internal/provider/state_upgraders*.go`
- `go.mod` / `go.sum` (new dependencies; note the SDK is v2, not the framework)
- `docs/**`, `examples/**` (hand-maintained; drift silently)

### Step 3: Categorize the Changes

| Category             | File patterns                            | What to check                                                         |
| -------------------- | ---------------------------------------- | --------------------------------------------------------------------- |
| Resources            | `internal/provider/resource_*.go`        | Schema correctness, `ForceNew`, CRUD, registration, deprecation       |
| Data sources         | `internal/provider/data_source_*.go`     | `Computed` shape, registration                                        |
| Provider wiring      | `internal/provider/provider.go`          | New entries in `ResourcesMap` / `DataSourcesMap`, `providerConfigure` |
| Shared constants     | `internal/provider/constants.go`         | Deprecation-message reuse                                             |
| State upgraders      | `internal/provider/state_upgraders*.go`  | `SchemaVersion` bump paired with an upgrader                          |
| Acceptance tests     | `internal/provider/*_test.go` (non-live) | `TF_ACC` gating, WireMock setup, attribute assertions                 |
| Live tests           | `internal/provider/*_live_test.go`       | `live_test,all` build tags, `...Live` / `...DriftDetection` naming    |
| Docs / examples      | `docs/**`, `examples/**`                 | Match current schema; run the `docs-drift` skill                      |
| Project rules/skills | `.claude/rules/**`, `.claude/skills/**`  | Frontmatter, path globs, trigger phrases                              |

### Step 4: Check Critical Requirements

**IMPORTANT: only review lines that were actually changed in the PR diff.** Context lines are for
understanding, not for review. Do not flag pre-existing issues in unchanged code.

#### 1. Registration completeness (MANDATORY for new resources/data sources)

- [ ] New resource has an entry in `ResourcesMap` (new data source → `DataSourcesMap`) in
      `provider.go`
- [ ] Type name is consistent across the map key, the file name, and the docs page

Red flags: a new `resource_*.go` with no `provider.go` change → the resource does not exist to
Terraform; a mismatched type name → broken resource address.

#### 2. Schema attribute correctness

- [ ] `Required` / `Optional` / `Computed` combination is valid and matches API semantics (no
      `Required`+`Computed`; server-assigned values are `Computed`; optional-with-default is
      `Optional`+`Default` or `Optional`+`Computed`)
- [ ] `ForceNew: true` on attributes the API cannot update in place — and _not_ on attributes it can
- [ ] `Sensitive: true` on secrets
- [ ] `ValidateFunc`/`ValidateDiagFunc` where values are constrained
- [ ] Nested blocks follow the sibling-resource `Elem` pattern

#### 3. Deprecation and backward compatibility

- [ ] No removed/renamed attribute or resource without a deprecation cycle
- [ ] Deprecations reuse a `constants.go` message (e.g. `deprecationMessageMajorRelease3`)
- [ ] No newly-added `ForceNew` or changed default that would recreate existing users' resources
      unexpectedly (unless gated on a major release and called out)

#### 4. State upgraders

- [ ] Any change to an existing resource's stored shape bumps `SchemaVersion` and adds a matching
      `StateUpgrader`. A shape change with no upgrader corrupts existing state on the next apply —
      treat a missing upgrader as a blocker.

#### 5. Testing

- [ ] New behavior has an acceptance test (`resource.Test`/`resource.ParallelTest`, `TF_ACC=1`),
      WireMock-backed where possible so it runs in CI without credentials
- [ ] The test asserts on the changed attributes, not merely that apply succeeds
- [ ] A live test is added/updated when real API behavior matters and WireMock can't cover it

### Step 5: Check Project-Specific Patterns

#### Docs drift (no `tfplugindocs` in CI)

- [ ] Schema changes are reflected in `docs/resources/*.md` / `docs/data-sources/*.md`, and
      `examples/` still validates. Invoke the `docs-drift` skill to compare docs/examples against the
      current Go schema maps.

#### Consistency with siblings

- [ ] A new resource follows the structure of the closest existing `resource_*.go` (CRUD function
      layout, error handling via `diag.Diagnostics`, import support, WireMock test scaffold). When in
      doubt, read the nearest sibling end to end before flagging a deviation.

### Step 6: Check PR Hygiene

- [ ] PR description fills in the template: release notes, What / Blast Radius (resources/data
      sources, `terraform import`), live-testing evidence, breaking-change status, feature-flag
      enablement
- [ ] One logical change per PR; no unrelated changes bundled in
- [ ] No secrets in the diff (`.tfvars` with real credentials, API keys, cluster IDs beyond fixtures)

## What NOT to Flag

### Tooling-owned style (avoid nitpicking)

- Formatting and import order — owned by `make checkfmt` (goimports). Don't hand-flag it.
- "This might fail CI" — the Semaphore build/test catches it; the author fixes it.

### Comment Preservation

- Never suggest deleting existing comments unless they are now actively misleading.
- Comments explain "why", not "what"; preserve or update them when code changes rather than removing.

## Output Format

### For Self-Review

```markdown
## Self-Review Summary

### Changes Overview

[Brief summary of what changed]

### Critical Requirements Checklist

- [ ] Registration (ResourcesMap / DataSourcesMap): [status, location of any gap]
- [ ] Schema correctness (Required/Optional/Computed, ForceNew, Sensitive): [status]
- [ ] Deprecation / backward compatibility: [status]
- [ ] SchemaVersion bump + state upgrader (if shape changed): [status]
- [ ] Acceptance/live test coverage: [status]

### Issues to Address Before PR

1. [High-priority issue with file:line]
2. [Medium-priority issue with file:line]

### Suggestions (Optional)

- [Nice-to-have improvements]

### Ready for Review?

[Yes / Not yet, with reasoning]
```

### For Formal Review

```markdown
## PR Review: #{number} - {title}

**Author:** {author}
**Branch:** {headRefName} → {baseRefName}
**Changes:** +{additions} / -{deletions} across {changedFiles} files

### Summary

[2-3 sentence summary of what the PR does and why]

### Changed Components

- [Categorized list of changed files, excluding generated/infra]

### Findings

#### Issues (Must Fix)

- [ ] **[category]**: [description] - `file:line`

#### Suggestions (Consider)

- [ ] **[category]**: [description] - `file:line`

#### Positive Observations

- [Good patterns, thorough tests, well-written code]

### Test Coverage Assessment

- **New tests added:** [Yes/No, list test files; WireMock vs live]
- **Coverage gaps:** [Untested attributes or edge cases]

### State & Compatibility Notes

[Any concern about forced recreation, missing state upgrader, or un-deprecated removal]

### Recommendation

**[APPROVE / REQUEST CHANGES / NEEDS DISCUSSION]**

[Brief rationale]
```

## Review Categories

Use these labels in findings:

| Category        | Description                                                                   |
| --------------- | ----------------------------------------------------------------------------- |
| `registration`  | Resource/data source not in `ResourcesMap`/`DataSourcesMap`, or name mismatch |
| `schema`        | Wrong `Required`/`Optional`/`Computed`, missing `ForceNew`/`Sensitive`        |
| `compatibility` | Un-deprecated removal/rename; unexpected forced recreation                    |
| `state`         | Stored-shape change with no `SchemaVersion` bump / state upgrader             |
| `testing`       | Missing acceptance/live coverage; test asserts nothing about the change       |
| `docs`          | Schema/doc/example drift; missing CHANGELOG entry                             |
| `secrets`       | Credentials or real IDs in the diff                                           |
| `consistency`   | Diverges from the closest sibling resource without reason                     |
| `style`         | Naming/conventions where `make checkfmt` does not enforce                     |

## Tips

- Start with the PR description and the PR template checklist.
- For a new resource, trace it end to end: constructor exists → registered in `provider.go` → schema
  attributes make sense → CRUD returns `diag.Diagnostics` → WireMock acceptance test present → docs
  page updated. A missing link usually means the resource is dead, unsafe, or untested.
- Use `Task` with the Explore agent for blast-radius questions (e.g. "which resources share this
  attribute before I change its `ForceNew`").
- Companion skill: `docs-drift` validates `docs/` and `examples/` against the current Go schema maps
  — run it whenever schema attributes change, since `tfplugindocs` is not in CI.
- When suggesting a simpler alternative, confirm it exists on terraform-plugin-sdk/v2 (this repo does
  not use the plugin framework) before posting.
