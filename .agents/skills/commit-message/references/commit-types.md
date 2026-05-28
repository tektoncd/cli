# Commit Types Reference

Complete reference for conventional commit types used in the Tekton CLI project.

## Scope and issue references

- Prefer **component names** as scope (e.g. `pipeline`, `task`, `log`, `actions`).
- To link work to a GitHub issue, add `Fixes #NNN` or `Closes #NNN` in the **commit body** (not the subject scope). Merging the PR then closes the issue automatically.

## Standard Types

### feat - New Features

Use for new features or functionality added to the codebase.

**Examples**:

- `feat(pipeline): add start --dry-run flag`
- `feat(task): add describe --output=json`
- `feat(plugins): add plugin discovery mechanism`
- `feat(flags): expose new --output format`

**When to use**:

- Adding new capabilities
- Introducing new commands or sub-commands
- Adding new CLI flags or output formats
- Adding new configuration options

### fix - Bug Fixes

Use for bug fixes that resolve incorrect behavior.

**Examples**:

- `fix(log): resolve streaming race condition`
- `fix(taskrun): correct describe output format`
- `fix(actions): handle missing resource gracefully`
- `fix(cli): prevent nil pointer on missing kubeconfig`

**When to use**:

- Fixing crashes or errors
- Resolving incorrect behavior
- Correcting logic errors

**CVE / security fixes**: Use scope `cve` and cite the advisory in the subject (see examples on `main`). Routine dependency bumps from Dependabot use `chore(deps):`, not `fix`.

### docs - Documentation

Use for documentation-only changes.

**Examples**:

- `docs(README): update installation steps`
- `docs(cmd): add pipeline start examples`
- `docs(AGENTS.md): update architecture conventions`
- `docs(contributing): add code review guidelines`

**When to use**:

- Updating README files
- Adding or improving code comments
- Updating CLI reference documentation
- Improving developer guides

### refactor - Code Refactoring

Use for code changes that neither fix bugs nor add features, but improve code structure.

**Examples**:

- `refactor(actions): extract common CRUD logic`
- `refactor(formatted): simplify table renderer`
- `refactor(cli): consolidate client setup`
- `refactor(cmd): reorganize package structure`

**When to use**:

- Improving code readability
- Simplifying complex logic
- Reorganizing code structure
- Extracting common functionality

**Must not** change observable behavior. If behavior changes, use `fix` (bug) or `feat` (new capability) instead.

### test - Testing

Use for adding or updating tests.

**Examples**:

- `test(pipeline): add describe unit tests`
- `test(taskrun): improve golden file coverage`
- `test(log): add streaming edge case tests`
- `test(e2e): add pipeline start E2E test`

**When to use**:

- Adding new test cases
- Improving test coverage
- Fixing flaky tests
- Adding integration or E2E tests

### chore - Maintenance Tasks

Use for routine maintenance tasks and tooling.

**Examples**:

- `chore(deps): update go dependencies`
- `chore(vendor): run go mod vendor`
- `chore(tools): update pre-commit hooks`
- `chore(golden): regenerate golden files`

**When to use**:

- Dependency updates
- Tooling configuration
- Repository maintenance
- Build script updates (minor)

### build - Build System

Use for changes to how the project is built (not routine dependency bumps — those are `chore(deps):`).

**Examples**:

- `build(Makefile): add vendor target`
- `build(goreleaser): update release config`
- `build(docker): update container base image`
- `build(go.mod): bump Go version to 1.24`

**When to use**:

- Changes to the root `Makefile` or build scripts
- Container or build image configuration
- GoReleaser configuration
- Bumping the Go version in `go.mod`

### ci - CI/CD Changes

Use for changes to continuous integration and release automation.

**Examples**:

- `ci(.github/workflows): add golangci-lint to CI`
- `ci(.github/workflows): update e2e matrix workflow`
- `ci(.tekton): update release pipeline`

**When to use**:

- Changes under `.github/workflows/`
- Changes under `.tekton/`

### perf - Performance Improvements

Use for changes that improve performance.

**Examples**:

- `perf(log): optimize pod log streaming`
- `perf(actions): reduce API call count`
- `perf(list): implement pagination`
- `perf(cli): lazy-load Tekton clients`

**When to use**:

- Optimization work
- Reducing latency
- Improving throughput
- Memory usage improvements

### style - Code Style

Use for formatting and style changes that don't affect code behavior.

**Examples**:

- `style(format): run goimports formatter`
- `style(lint): fix golangci-lint warnings`
- `style(markdown): fix markdownlint issues`
- `style(naming): apply consistent naming`

**When to use**:

- Running code formatters
- Fixing linter warnings (style-only)
- Applying consistent naming
- Whitespace/indentation fixes

### revert - Revert Previous Commit

Use for reverting previous commits.

**Examples**:

- `revert: undo breaking CLI flag change`
- `revert(pipeline): revert start refactoring`
- `revert: "feat(task): add new flag"`

**When to use**:

- Undoing problematic changes
- Rolling back breaking changes
- Reverting commits that caused issues

**Format**: Include reference to original commit in body

## Breaking Changes

For any commit type, add `!` after type/scope to indicate breaking change:

**Examples**:

- `feat(cli)!: change default output format`
- `fix(flags)!: remove deprecated --namespace flag`
- `refactor(actions)!: restructure client interface`

**Body should include**:

```markdown
BREAKING CHANGE: <description of breaking change and migration path>
```

## Type Selection Guide

### Decision Tree

1. **Does it add new functionality?** → `feat`
2. **Does it fix a bug?** → `fix`
3. **Is it documentation only?** → `docs`
4. **Does it change code structure without behavior change?** → `refactor`
5. **Is it test-related?** → `test`
6. **Is it dependency/maintenance?** → `chore`
7. **Is it build system related?** → `build`
8. **Is it CI/CD related?** → `ci`
9. **Does it improve performance?** → `perf`
10. **Is it formatting/style only?** → `style`
11. **Does it revert a previous commit?** → `revert`

### Common Mistakes

**Wrong**: `chore(cli): add new subcommand`
**Right**: `feat(cli): add new subcommand`
*Reason: Adding a subcommand is a feature, not maintenance*

**Wrong**: `feat(tests): add test coverage`
**Right**: `test(pipeline): add test coverage`
*Reason: Tests have their own type*

**Wrong**: `fix(docs): update README`
**Right**: `docs(README): update installation steps`
*Reason: Documentation has its own type*

**Wrong**: `chore(ci): update GitHub Actions workflow`
**Right**: `ci(.github/workflows): update test workflow`
*Reason: CI changes have their own type; scope the path touched*

## Multiple Changes in One Commit

When a commit includes multiple types of changes, choose the most significant:

**Example**: Adding feature + tests

- **Choose**: `feat(pipeline): add start --dry-run flag`
- **Body mentions**: "Includes unit tests"

**Example**: Bug fix + refactoring

- **Choose**: `fix(log): resolve streaming race condition`
- **Body mentions**: "Refactored log reader for clarity"

**Guideline**: If changes are too diverse, consider splitting into multiple commits.
