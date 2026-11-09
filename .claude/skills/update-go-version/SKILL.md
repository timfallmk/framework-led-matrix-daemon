---
name: update-go-version
description: Update every Go version reference in this repo (go.mod, Dockerfile, CI workflows, README, CONTRIBUTING) to the latest stable Go release, optionally upgrading dependencies and opening a PR.
when_to_use: >-
  When asked to bump, upgrade, or update Go, golang, or the Go toolchain; when go.mod,
  the Dockerfile, CI workflows, or docs disagree on the Go version; or when CI or a
  Docker build fails because the Go version is outdated or mismatched. Pass `deps` if
  dependency upgrades were also requested and `pr` only if a commit/PR was requested.
argument-hint: "[deps] [pr]"
context: fork
background: false
effort: medium
allowed-tools: >-
  Read Edit
  Bash(curl -fsS https://go.dev/VERSION*)
  Bash(grep *) Bash(git grep *) Bash(git status *) Bash(git diff *)
  Bash(go mod edit -go=*) Bash(go mod tidy) Bash(go get -u ./...) Bash(go build ./...) Bash(make test)
---

# Update Go version

Arguments: `$ARGUMENTS`
- `deps`: also upgrade all module dependencies.
- `pr`: commit the changes, push, and open a PR.

## Versions

Latest stable release (first line; strip the `go` prefix):

!`curl -fsS 'https://go.dev/VERSION?m=text'`

Current `go` directive in go.mod:

!`grep -m1 '^go ' ${CLAUDE_PROJECT_DIR}/go.mod`

If the current version already equals the latest, and `deps` wasn't passed, report "Already up to date" and stop.

Below, `NEW` is the latest full version (for example `1.27.3`) and `NEW_MINOR` is its major.minor (`1.27`). `OLD` and `OLD_MINOR` are the same for the current version.

## 1. Update references

Full patch version (`NEW`):

| File | Pattern |
|------|---------|
| `go.mod` | `go X.Y.Z` |
| `Dockerfile` | `FROM golang:X.Y.Z-alpine AS builder` |
| `.github/workflows/ci.yml` | `GO_VERSION: "X.Y.Z"` and the `go-version: ['X.Y.Z']` matrix entry |
| `.github/workflows/release.yml` | `GO_VERSION: 'X.Y.Z'` |

Minor version (`NEW_MINOR`):

| File | Pattern |
|------|---------|
| `README.md` | `Go X.Y or later (for building)` and `# Install Go X.Y+` |
| `CONTRIBUTING.md` | `**Go X.Y** or later` |

Update `go.mod` with `go mod edit -go=NEW`, not by hand. The CI or Dockerfile may already be ahead of `go.mod`; update whatever is still behind.

## 2. Dependencies (only with `deps`)

Run `go get -u ./...`.

## 3. Tidy and verify

1. `go mod tidy`
2. `git grep -nE "OLD|Go OLD_MINOR" -- ':!go.sum' ':!.claude'`, with the real versions and the dots escaped. Fix anything it finds that refers to the Go toolchain. Ignore unrelated matches, such as a dependency version that happens to contain the same digits.
3. `go build ./...` and `make test`. If either fails, stop and report the failure output. Don't try to fix code beyond what the version bump needs.

## 4. Branch, commit, PR (only with `pr`)

1. If the current branch is `main`, run `gh tidy`, then `git switch -c tfall/bump-go-NEW`. Otherwise stay on the current branch.
2. Commit with the message `Bump Go to NEW` (add `and upgrade deps` when `deps` was passed), ending with:
   `Co-Authored-By: Claude <noreply@anthropic.com>`
3. Push and open a PR against `main` with `gh pr create`. Its body summarizes the changes and ends with:
   `🤖 Generated with [Claude Code](https://claude.com/claude-code)`

## Report

Return the old and new versions, each file changed and what changed in it, whether dependencies were upgraded, the build and test results, and the PR URL if one was opened.
