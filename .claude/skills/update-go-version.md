# Update Go Version

## Skill Configuration

- **Name**: update-go-version
- **Description**: Check for the latest Go release and update all version references across the repository
- **User-invocable**: true
- **Trigger**: /update-go-version

## Instructions

When invoked, perform a comprehensive Go version update across the entire repository:

### 1. Check Latest Go Version

Search the web and fetch https://go.dev/dl/ to determine the current latest stable Go release version.

### 2. Read Current Version

Read `go.mod` to determine the current Go version in use.

### 3. Compare Versions

If the repo is already on the latest version, inform the user and stop. Otherwise, proceed with updates.

### 4. Update All References

Update Go version references in ALL of these locations:

**Build files (use full patch version, e.g., `1.26.0`):**
- `go.mod` - the `go` directive
- `Dockerfile` - the `golang:X.Y.Z-alpine` builder image

**CI/CD workflows (use full patch version):**
- `.github/workflows/ci.yml` - `GO_VERSION` env var and `go-version` matrix
- `.github/workflows/release.yml` - `GO_VERSION` env var

**Documentation (use minor version, e.g., `1.26`):**
- `README.md` - minimum version requirements and install instructions
- `CONTRIBUTING.md` - prerequisites section

### 5. Run Grep Safety Check

After making changes, grep the entire repository for the OLD version string to ensure nothing was missed. The old version could appear as:
- The full patch version (e.g., `1.25.4`)
- The minor version in docs (e.g., `1.25`)

Exclude `go.sum`, `.git/`, and binary files from this check.

### 6. Tidy Modules

Run `go mod tidy` to ensure module consistency after the version bump.

### 7. Report Changes

Summarize all files changed and the old/new versions for the user.

### 8. Branch, Commit, and PR (if requested)

If the user requests it:
1. Create a new branch: `chore/update-go-<new-version>`
2. Stage and commit all changes with message: `chore: update Go version from <old> to <new>`
3. Push and open a PR
