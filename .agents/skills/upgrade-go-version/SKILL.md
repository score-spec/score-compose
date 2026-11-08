---
name: upgrade-go-version
description: Upgrade the Go version of this repository (go.mod `go` and `toolchain` directives and the Go base image in the Dockerfile), update all Go dependencies, and open a draft pull request. Use when a maintainer asks to bump, upgrade or update Go, the Go toolchain, or Go dependencies to a new Go release (e.g. "upgrade to Go 1.27.4"). Replaces the former `upgrade-go-version` GitHub Actions workflow.
---

# Upgrade Go version

Bump the Go version in `go.mod`, refresh every Go dependency, and open a **draft** PR for a maintainer to review.

## Inputs

| Input | Example | Notes |
|-------|---------|-------|
| `go-version` | `1.27` | Language version for the `go` directive in `go.mod` (`major.minor`). |
| `go-toolchain-version` | `1.27.3` | Full toolchain release for the `toolchain` directive (`major.minor.patch`, **without** the `go` prefix). |

If the user did not provide them, look up the latest stable release and **confirm with the user before continuing**:

```bash
curl -s 'https://go.dev/dl/?mode=json' | jq -r '[.[] | select(.stable)][0].version'
# -> go1.27.3  => toolchain-version=1.27.3, go-version=1.27
```

Validate both inputs before using them in any command. They must match these patterns, otherwise stop and ask:

- `go-version`: `^[0-9]+\.[0-9]+$`
- `go-toolchain-version`: `^[0-9]+\.[0-9]+\.[0-9]+$`

## Prerequisites

- `go`, `git` and `gh` (authenticated: `gh auth status`) are installed.
- `docker` is installed and logged in to `dhi.io` (`docker login dhi.io`), needed to resolve the new base image digest for the `Dockerfile`.
- The working tree is clean (`git status --porcelain` prints nothing). If not, stop and tell the user.
- Git `user.name` / `user.email` are set (commits must be signed off, see DCO in `CONTRIBUTING.md`).

## Steps

1. **Start from an up-to-date `main`.**

   ```bash
   git fetch origin main
   git switch -C upgrade-go-<go-toolchain-version> origin/main
   ```

   The branch is named after the toolchain version, e.g. `upgrade-go-1.27.3`. `-C` resets it if a previous attempt left one behind locally.

2. **Use the target toolchain**, so `go get -u all` resolves against the new Go version. Go downloads it automatically when needed.

   ```bash
   export GOTOOLCHAIN=go<go-toolchain-version>
   go version   # must print go<go-toolchain-version>
   ```

3. **Update `go.mod` and all dependencies.**

   ```bash
   go mod edit -go <go-version>
   go mod edit -toolchain go<go-toolchain-version>
   go get -u all
   go mod tidy
   ```

4. **Update the Go image in the `Dockerfile`.** The first line pins the builder image by tag and digest, e.g. `FROM dhi.io/golang:1.27.3-alpine3.24-dev@sha256:... AS builder`. Bump the Go version in the tag, keep the rest of the tag (Alpine version, `-dev` suffix) and replace the digest with the one for the new tag.

   ```bash
   # Current tag, e.g. 1.27.3-alpine3.24-dev
   old_tag=$(sed -nE '1s#^FROM dhi\.io/golang:([^@ ]+).*#\1#p' Dockerfile)
   # Swap the leading version, keep the suffix
   new_tag="<go-toolchain-version>-${old_tag#*-}"

   # Resolve the multi-arch index digest (needs `docker login dhi.io`)
   digest=$(docker buildx imagetools inspect "dhi.io/golang:${new_tag}" --format '{{.Manifest.Digest}}')
   echo "$digest" | grep -Eq '^sha256:[0-9a-f]{64}$' || { echo "could not resolve digest"; exit 1; }

   sed -i -E "1s#^FROM dhi\.io/golang:[^ ]+#FROM dhi.io/golang:${new_tag}@${digest}#" Dockerfile
   ```

   If the new tag does not exist yet (the image is sometimes published a little after the Go release) or the Alpine version in the suffix is no longer offered, **stop and ask the user** instead of guessing a tag. Never write a digest you did not resolve. If `docker` or the `dhi.io` login is unavailable, tell the user and ask whether to continue without the `Dockerfile` change.

5. **Verify** the result before committing. The old workflow relied on PR CI for this; doing it locally avoids opening a PR that is obviously broken.

   ```bash
   go build ./...
   make test
   docker build -t score-compose:upgrade-go .   # only if Docker is available; confirms the new base image works
   ```

   If the build or tests fail because of an upgraded dependency, report the failure (with the output) to the user and ask how to proceed. Do not silently pin dependencies back or edit code to make it pass.

6. **Review the diff.** `git diff --stat` should only touch `go.mod`, `go.sum` and `Dockerfile`. Summarize the notable changes for the user: the new `go`/`toolchain` lines, the new image tag, and which direct dependencies moved major or minor versions. CI workflows read the version from `go.mod` (`go-version-file`), so they need no change; if you spot any other hard-coded Go version (`grep -rnE 'go ?1\.[0-9]+' --include='*.yaml' --include='*.yml' --include=Makefile .`), report it to the user rather than editing it.

7. **Commit with a DCO sign-off.**

   ```bash
   git add go.mod go.sum Dockerfile
   git commit -s -m "Upgrade Go version to <go-toolchain-version>"
   ```

8. **Ask the user for confirmation, then push and open a draft PR.** Pushing and opening a PR is outward-facing: show the branch, the commit and the diff summary first and wait for a clear yes.

   ```bash
   git push --force-with-lease -u origin upgrade-go-<go-toolchain-version>
   gh pr create --draft \
     --base main --head upgrade-go-<go-toolchain-version> \
     --title "Upgrade Go version to <go-toolchain-version>" \
     --body "$(cat <<'EOF'
   Upgrade Go version to <go-toolchain-version>:
   https://go.dev/doc/devel/release
   - [ ] Set PR as "Ready for review"
   - [ ] Check CI
   - [ ] Submit PR review
   EOF
   )"
   ```

   Replace `<go-toolchain-version>` in the branch name, title and body with the real value. If an open PR from this branch already exists (`gh pr list --head upgrade-go-<go-toolchain-version> --state open`), the push updates it; do not open a second one.

9. **Report** the PR URL to the user.

## Why the PR is a draft

PRs created with the default `GITHUB_TOKEN` do not trigger CI workflows. Opening the PR as a draft and asking a maintainer to mark it "Ready for review" (or push an empty commit) gets CI to run. When this skill pushes with the maintainer's own `gh` credentials CI would normally trigger anyway, but the draft + checklist keeps the review process identical to the previous workflow.

## Don'ts

- Do not push to `main` or merge the PR.
- Do not run this on a dirty working tree or from a fork without confirming the target remote.
- Do not pass unvalidated user input into shell commands.
