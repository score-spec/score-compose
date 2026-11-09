# AGENTS.md

Guidance for AI coding agents working in `score-compose`. Humans should start with [README.md](README.md) and
[CONTRIBUTING.md](CONTRIBUTING.md); this file captures what an agent needs to be productive and avoid common mistakes.

## What this project is

`score-compose` is a Go CLI that converts a [Score](https://score.dev) workload specification (`score.yaml`) into a
Docker Compose file (`compose.yaml`). It is the Docker Compose implementation of the Score spec, part of the
[`score-spec`](https://github.com/score-spec) organisation (sibling projects: `score-go`, `score-k8s`, `spec`).

The CLI has three main workflows:

- `score-compose init` creates a `.score-compose/` state directory (and a sample `score.yaml`) in the current directory.
- `score-compose generate score.yaml ...` converts one or more Score files into `compose.yaml`, provisioning any
  declared resources along the way.
- `score-compose provisioners list` and `score-compose resources list` / `resources get-outputs` inspect the configured
  provisioners and the resources recorded in state.

## Repository layout

| Path | Purpose |
|------|---------|
| `cmd/score-compose/main.go` | Thin entrypoint; calls `command.Execute()` and prints errors to stderr. |
| `internal/command/` | Cobra commands: `init`, `generate`, `provisioners`, `resources`, `version`, plus the root command and logging flags. |
| `internal/command/default.provisioners.yaml` | The built-in provisioners (volume, service-port, redis, postgres, mysql, s3, dns, route, amqp, mongodb, kafka-topic, elasticsearch, mssql, environment). Embedded with `go:embed` and written to `.score-compose/zz-default.provisioners.yaml` by `init`. |
| `internal/compose/` | Converts a Score workload into compose services/volumes/networks and writes the compose file. |
| `internal/provisioners/` | Provisioner interface and the resource provisioning loop (`core.go`). Implementations: `templateprov/` (Go text/template + sprig), `cmdprov/` (external command), `envprov/` (environment variables). `loader/` parses `*.provisioners.yaml` files and dispatches on the URI scheme (`template://`, `cmd://`, `local-env://`). |
| `internal/project/` | The persisted state (`.score-compose/state.yaml`) built on `score-go`'s `framework.State`. |
| `internal/patching/` | Patch templates applied to the generated compose output. |
| `internal/util/`, `internal/logging/`, `internal/version/` | Shared helpers, slog handler, and build version info (set via `-ldflags`). |
| `examples/` | Numbered example workloads (`01-hello` ... `16-patching-templates`), each with a README and `score.yaml`. These are exercised by tests. |
| `.github/workflows/` | CI (`ci.yaml`), release, scorecard, and automated Go-version upgrades. |

All application code lives under `internal/`; there is no public Go API. Keep it that way unless a maintainer
asks otherwise.

## Build, test, and verify

Requires Go (version pinned in `go.mod`: currently `go 1.27` / `toolchain go1.27.3`). Docker with the Compose plugin is
needed only for the full test suite.

```bash
make build                         # go build ./cmd/score-compose/  -> ./score-compose
make test                          # go vet ./... && go test ./... -cover -race
NO_DOCKER=true go test ./... -race # skip the `docker compose config` validation (no Docker needed, faster)
go test ./internal/command -run TestExample/01-hello   # run a single test
```

Notes:

- **`make test` runs `go env -w GOTOOLCHAIN=...`**, which changes the user's global Go environment. Prefer running
  `go vet ./... && go test ./... -cover -race` directly when working on someone's machine or in a shared sandbox.
- Many tests shell out to `docker compose -f compose.yaml config --quiet --dry-run` to validate generated output.
  They are skipped when `NO_DOCKER` is set to any non-empty value.
- CI runs `go vet ./...`, `go test ./... -cover -race`, a goreleaser snapshot build, and the license-header check
  (below). Make sure all of these pass locally before proposing a change.
- Dockerfile builds use `dhi.io` (Docker Hardened Images) base images; building the image locally needs access to that
  registry. Don't change the base images or their digests unless that is the task.

### License headers

Every `.go` file under `cmd/` and `internal/` must start with the Apache 2.0 header (see any existing file, copyright
holder `The Score Authors`). Check with:

```bash
go install github.com/google/addlicense@latest
addlicense -l apache -check -v -ignore '**/*.yaml' -c 'The Score Authors' ./cmd ./internal/
```

Drop `-check` to add missing headers.

## Testing conventions

- Tests use `testing` with `github.com/stretchr/testify` (`assert` / `require`). Table-driven tests are the norm.
- Command tests live in `internal/command/*_test.go`, run the real cobra root command via the
  `executeAndResetCommand` helper, and `os.Chdir` into a temp dir. Don't run them in parallel and don't assume a working
  directory.
- `internal/command/generate_examples_test.go` runs `init` + `generate` against `examples/*` and compares the full
  `compose.yaml` output. **If you change an example's `score.yaml`, a default provisioner, or conversion behaviour, the
  expected output in this test must be updated too**, and the example's README should stay consistent.
- Golden files live in `internal/command/testdata/` and `internal/command/fixtures/`. `provisioners_test.go` supports
  regenerating them with `go test ./internal/command -run <Test> -update`; review the diff, don't commit blindly.
- Bug fixes and new features need tests (a CONTRIBUTING requirement).

## Code conventions

- Standard Go style: run `gofmt` (and `goimports` ordering: stdlib, third-party, then `github.com/score-spec/score-compose/...`
  in separate groups, as in existing files) and keep `go vet` clean.
- Logging uses `log/slog`. User-facing errors are returned (wrapped with `fmt.Errorf("...: %w", err)`) and printed once
  in `main()`; commands set `SilenceErrors: true`, so don't print errors inside commands. Output meant for the user goes
  to stdout; logs go to stderr.
- The Score spec types come from `github.com/score-spec/score-go` and compose types from
  `github.com/compose-spec/compose-go/v2`. Reuse them rather than defining parallel structs.
- State written to `.score-compose/state.yaml` is decoded with `KnownFields(true)` and written atomically
  (temp file + rename). Adding or renaming fields in `internal/project` affects users' existing state directories, so
  treat it as a compatibility-sensitive change.
- Generated compose output must be deterministic (sorted keys / stable ordering); tests compare it byte-for-byte.
- Default provisioners are user-visible API. Changing a provisioner's outputs, params or generated service names can
  break existing users. When you edit `default.provisioners.yaml`, also update the resource table in
  [README.md](README.md) and the matching example under `examples/`.
- Provisioner templates run with the [sprig](https://masterminds.github.io/sprig/) function set (including `fail`); keep templates pure and rely on `state`/`init` for anything
  random (e.g. passwords) so regenerating is idempotent.

## Things to avoid

- Don't commit generated artifacts: `.score-compose/`, `compose.yaml`, built `score-compose` binaries, coverage output
  (all in `.gitignore`).
- Don't hand-edit `go.sum`; use `go mod tidy`. Dependency bumps are normally handled by Dependabot (weekly) and Go
  version bumps by the `upgrade-go-version` workflow, so avoid unrelated dependency changes in feature PRs.
- Don't unpin GitHub Actions: workflow actions and base images are pinned by commit SHA / digest on purpose
  (OpenSSF Scorecard). Update them via the existing automation.
- Don't add new top-level CLI flags or commands, or change flag semantics, without discussing it with maintainers; the CLI
  surface is documented externally at [docs.score.dev](https://docs.score.dev/docs/score-implementation/score-compose/).
- Don't modify `LICENSE`, `NOTICE`, `CODEOWNERS`, `MAINTAINERS.md`, or `SECURITY.md` unless explicitly asked.
- Never run `generate`/`init` in the repo root while testing manually; use a scratch directory so you don't leave
  state files behind.

## Commits and pull requests

- Branch from `main`, open PRs against `main`, and rebase on `main` before submitting.
- **Every commit must be signed off** (DCO): use `git commit -s`. The sign-off email must match the commit author.
- Keep commits small. The first line says what changed; the body explains why and how (sharing command output is
  encouraged). Recent history uses plain imperative titles and `chore(deps): ...` / `refactor: ...` prefixes, with the PR
  number appended by GitHub.
- Fill in [.github/pull_request_template.md](.github/pull_request_template.md): description, what the PR does, type of
  change, and whether documentation was updated.
- If AI tools helped prepare the PR, disclose it in the description (see [AI-assisted contributions](#ai-assisted-contributions)).
- Large or high-impact changes should be discussed with maintainers in an issue first (or in `#score` on the CNCF Slack).
- Merging requires green CI and approval from a maintainer.

## AI-assisted contributions

AI tools may be used to help write a PR, but the human author is fully responsible for it. If you are an agent working
on behalf of a contributor, help them meet these project rules:

- **Disclose AI use in the PR description.** Any PR prepared with AI assistance must say so. A sentence like "This PR was
  written in part with the assistance of generative AI." is sufficient. When you draft a PR description, include it.
- **The author must understand every change.** Keep changes small and explainable. Reviewers may ask why something was
  done, and a PR is closed if the author cannot explain it.
- **Verify before submitting.** Don't leave the first review of AI-generated changes to the maintainers. Run the checks
  in [Build, test, and verify](#build-test-and-verify), review the diff yourself, and tell the contributor what you did
  and did not verify. Do not claim tests pass unless you ran them.
- **Follow the contribution policies.** All commits must follow [CONTRIBUTING.md](CONTRIBUTING.md), including DCO
  sign-off and the commit message guidance above.
- **Do not answer reviewers with AI.** Replies to review comments must come from the human author, not from generated
  text. Don't draft, post or auto-respond to review comments on a contributor's behalf. Reviewers want to engage with the
  person directly, and a PR is closed if the author does not engage.

## Releases

Releases are cut by maintainers by pushing a `0.*` tag; `release.yaml` runs GoReleaser (binaries, Homebrew tap) and
builds/signs the container image. Version info is injected with `-ldflags` into `internal/version`. Agents should not
create tags or releases.

## Useful references

- Documentation: <https://docs.score.dev/docs>
- Score specification: <https://github.com/score-spec/spec>
- Shared Go library: <https://github.com/score-spec/score-go>
- Security reporting: see [SECURITY.md](SECURITY.md)
