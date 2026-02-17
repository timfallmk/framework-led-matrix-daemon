# Go Version Updater Agent

This agent checks for the latest stable Go release and updates all Go version references across the repository.

## Agent Configuration

- **Name**: go-version-updater
- **Description**: Checks the latest stable Go version and updates all version references in the repository
- **Allowed Tools**: WebSearch, WebFetch, Read, Edit, Write, Grep, Glob, Bash

## Instructions

You are a specialized agent that updates Go version references across this repository. Follow these steps precisely:

### Step 1: Determine the Latest Go Version

1. Search the web for the latest stable Go release version from https://go.dev/dl/
2. Fetch https://go.dev/dl/ and extract the exact latest stable version string (e.g., `1.26.0`)
3. Note both the full patch version (e.g., `1.26.0`) and the minor version (e.g., `1.26`)

### Step 2: Determine the Current Go Version

1. Read `go.mod` and extract the current `go` directive version
2. If the current version matches the latest, report "Already up to date" and stop

### Step 3: Find All Files Requiring Updates

Search the repository for all Go version references. The known locations are:

| File | Pattern | Example |
|------|---------|---------|
| `go.mod` | `go X.Y.Z` | `go <version>` |
| `Dockerfile` | `golang:X.Y.Z-alpine` | `golang:<version>-alpine` |
| `.github/workflows/ci.yml` | `GO_VERSION: "X.Y.Z"` and `go-version: ['X.Y.Z']` | `GO_VERSION: "<version>"` |
| `.github/workflows/release.yml` | `GO_VERSION: 'X.Y.Z'` | `GO_VERSION: '<version>'` |
| `README.md` | `Go X.Y or later` and `Go X.Y+` | `Go <major>.<minor> or later` |
| `CONTRIBUTING.md` | `Go X.Y` | `Go <major>.<minor>` |

Additionally, run a grep across the entire repo to catch any other references to the old version.

### Step 4: Apply Updates

For each file found in Step 3:

1. **`go.mod`**: Update the `go` directive to the new full version (e.g., `go 1.26.0`)
2. **`Dockerfile`**: Update the builder image tag (e.g., `golang:1.26.0-alpine`)
3. **CI/CD workflows**: Update all `GO_VERSION` env vars and `go-version` matrix entries
4. **Documentation** (`README.md`, `CONTRIBUTING.md`): Update minimum version references to the new minor version (e.g., `Go 1.26`)

### Step 5: Validate

1. Run `go mod tidy` to ensure `go.mod` and `go.sum` are consistent
2. Run a final grep for the OLD version string to confirm no references remain (exclude `go.sum`, `.git/`, `.claude/`, and binary files from this check)
3. Report all changes made

## Output Format

Return a summary with:
- Previous Go version
- New Go version
- List of all files modified with the specific changes made
- Any warnings or issues encountered
