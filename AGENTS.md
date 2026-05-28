# Tekton CLI (`tkn`)

Command-line client for [Tekton Pipelines](https://tekton.dev). Provides
sub-commands to create, list, describe, start, cancel, delete, and log
Tekton resources (Tasks, Pipelines, PipelineRuns, etc.).

---

## Build & Test Commands

```bash
# Build
go build ./cmd/tkn
make bin/tkn

# Cross-platform binaries
make cross

# Test — runs without a cluster
make test
make test-unit
make test-unit-race

# Lint — must pass before every PR
make lint              # golangci-lint + goimports + yamllint
make lint-go           # Go only, all packages
make lint-go PKG=./pkg/pods/...   # single package (fast)

# Code formatting
make fmt               # gofmt
make goimports         # goimports

# Update golden files (test snapshots)
make update-golden

# Regenerate docs and formatting
make generated         # update-golden + docs + fmt

# Dependency update — required after go.mod changes
go mod vendor
```

E2E tests require a live cluster with Tekton Pipelines installed.
See [DEVELOPMENT.md](./DEVELOPMENT.md) for cluster setup.

---

## Key Conventions

1. **Cobra command tree.** Every user-facing command lives under `pkg/cmd/`.
   The root command is wired in `cmd/tkn/main.go`. Each resource type
   (pipeline, task, pipelinerun, …) has its own sub-package with standard
   verbs: `create`, `delete`, `describe`, `list`, `start`, `logs`.

2. **`pkg/` holds the shared logic.** Packages like `pkg/actions/`,
   `pkg/formatted/`, `pkg/log/`, and `pkg/pods/` do the real work (API calls,
   output formatting, log streaming). The command files in `pkg/cmd/` are mostly wiring, and should
   only handle flag parsing and call into these packages — don't put core
   logic directly in command handlers.

3. **Golden-file tests.** Many commands use golden files for expected output.
   After changing command output, run `make update-golden` and commit the
   updated `.golden` files.

4. **Vendored dependencies.** This project vendors (`go mod vendor`).
   Always commit `vendor/` changes when updating `go.mod`.

5. **`-mod=vendor` everywhere.** Builds and lints use `-mod=vendor`.
   Do not remove this flag.

6. **Version injection via ldflags.** The client version is injected at build
   time through `LDFLAGS`. See the `Makefile` for `VERSIONLDFLAG`.

---

## Architecture

```
cmd/
  tkn/          # main entry point
  docs/         # doc generation binary

pkg/
  cmd/          # Cobra command implementations (one sub-package per resource)
    pipeline/   # tkn pipeline create|delete|describe|list|start|logs
    task/       # tkn task create|delete|describe|list|start|logs
    pipelinerun/
    taskrun/
    ...
  actions/      # generic Tekton resource CRUD via dynamic client
  cli/          # shared CLI params, streams, clients
  flags/        # common flag definitions
  formatted/    # output formatters (table, YAML, JSON)
  log/          # log streaming for PipelineRun/TaskRun
  pods/         # logic for finding the right container in a pod, pod/container log helpers
  params/       # parameter parsing and merging
  plugins/      # tkn plugin discovery
  version/      # client/server version reporting

docs/           # auto-generated CLI reference (markdown + man pages)
test/           # E2E test framework and scripts
hack/           # helper scripts (golden-file updates, etc.)
third_party/    # vendored non-Go assets
```

**Key patterns:**

- Each command sub-package typically has: `<verb>.go` (command definition),
  `<verb>_test.go` (unit tests), and `testdata/` with `.golden` files.
- `pkg/cli/Clients` bundles the Tekton, Kubernetes, and dynamic clients.
  Commands receive this via their `Params` interface.

---

## PR Conventions

- Pull requests must follow the repository PR template defined in `.github/pull_request_template.md`.
- `make lint` must pass with zero issues.
- `make test` must pass with zero failures.
- Run `make generated` and commit generated/golden files when command output changes.
- Commit messages follow [Tekton community standards](https://github.com/tektoncd/community/blob/master/standards.md#commit-messages).

---

## Windows checkout

`CLAUDE.md` points to `AGENTS.md`, and `.claude/skills` points to `.agents/skills`.
This works on Linux, macOS, and GitHub; on Windows, enable symlinks when cloning:

```bash
git clone -c core.symlinks=true https://github.com/tektoncd/cli.git
```

Alternatively, set `core.symlinks=true` in your git config before checkout.

---

## Skills

For complex workflows, use these repo-local skills:

- **Commit messages**: Conventional commits with component scopes, line length
  validation, DCO Signed-off-by, and Assisted-by trailers. Trigger: "create
  commit", "commit changes", "generate commit message"
- **Release notes**: Gather PRs between tags, categorize, output formatted
  markdown, optionally update GitHub release. Trigger: "create release note",
  "generate release notes", "release changelog"
