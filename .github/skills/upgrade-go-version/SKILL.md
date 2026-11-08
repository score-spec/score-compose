---
name: upgrade-go-version
description: Upgrade the Go language and toolchain versions and Go module dependencies in score-compose.
---

# Upgrade Go version

Use this skill when asked to upgrade the Go version or toolchain in this repository.

1. Obtain both the Go language version (for example, `1.27`) and the Go toolchain version (for example, `1.27.3`) from the request. Ask for any missing version rather than guessing. The toolchain version must be compatible with the language version.
2. Check the working tree and preserve any existing changes. Ensure the requested toolchain is available, or that Go's automatic toolchain download can obtain it.
3. From the repository root, run the following commands with the requested versions substituted. Set `GOTOOLCHAIN` for each command so the requested toolchain is used as the starting toolchain, with automatic upgrades enabled if required by dependencies.

   ```sh
   GOTOOLCHAIN=go<toolchain-version>+auto go mod edit -go <go-version>
   GOTOOLCHAIN=go<toolchain-version>+auto go mod edit -toolchain go<toolchain-version>
   GOTOOLCHAIN=go<toolchain-version>+auto go get -u all
   GOTOOLCHAIN=go<toolchain-version>+auto go mod tidy
   ```

4. Review the resulting `go.mod` and `go.sum` changes, including whether dependency upgrades changed the requested Go or toolchain directives. Check whether the pinned Go builder image in `Dockerfile` remains compatible. Report any mismatch rather than silently claiming the requested versions were applied.
5. Run `go vet ./...` and `go test ./... -cover -race` with the selected toolchain. If Docker is unavailable, the tests can be run with `NO_DOCKER=true`; report that Docker-based validation was skipped. Report any failures or checks that could not be run.
6. If asked or authorized to submit the change, create a signed-off commit and a draft pull request titled `Upgrade Go version to <toolchain-version>`. Follow the repository's pull request template and include the Go release notes at https://go.dev/doc/devel/release, the validation results, and the following follow-up checklist:

   - [ ] Set PR as "Ready for review"
   - [ ] Check CI
   - [ ] Submit PR review

   Follow the agent's applicable instructions for branches, commits, pushes, and pull request creation. Do not overwrite an existing branch or include unrelated changes.
