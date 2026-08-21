---
name: commit-message
description: This skill should be used when the user asks to "create a commit", "generate commit message", "commit changes", "make a commit", mentions "conventional commits", or discusses commit message formatting. Provides guided workflow for creating properly formatted commit messages with line length validation and required trailers.
version: 0.2.0
---

# Conventional Commit Message Creation

Create properly formatted conventional commit messages following project standards with line length validation and required trailers.

## Purpose

Generate commit messages that:

- Follow conventional commits format (`type(scope): description`)
- Use component names or GitHub issue numbers as scope
- Respect line length limits (50 for subject, 72 for body)
- Include required trailers (Signed-off-by, Assisted-by)
- Match [Tekton commit message standards](https://github.com/tektoncd/community/blob/master/standards.md#commit-messages)

## Quick Workflow

1. **Analyze changes**: Run git status and git diff to understand modifications
2. **Determine scope**: Use component name from changed files, or GitHub issue number if available
3. **Generate message**: Create conventional commit message with proper formatting
4. **Add trailers**: Include Signed-off-by and Assisted-by trailers
5. **Confirm with user**: Display message and wait for approval before committing

**CRITICAL**: Never commit without explicit user confirmation.

## Conventional Commit Format

### Structure

```text
<type>(<scope>): <description>

[optional body]

Signed-off-by: <name> <email>
Assisted-by: <Model name> (via <Tool>)
```

### Type Selection

Choose the appropriate commit type based on changes:

| Type | Description | Example |
| ------ | ------------- | --------- |
| `feat` | New features | `feat(pipeline): add start --output flag` |
| `fix` | Bug fixes | `fix(taskrun): resolve log streaming race` |
| `docs` | Documentation | `docs(README): update installation steps` |
| `refactor` | Code refactoring | `refactor(actions): simplify CRUD helpers` |
| `test` | Test changes | `test(pipelinerun): add describe unit tests` |
| `chore` | Maintenance | `chore(deps): update go dependencies` |
| `build` | Build system | `build(Makefile): add vendor target` |
| `ci` | CI/CD changes | `ci(github): add golangci-lint action` |
| `perf` | Performance | `perf(log): optimize pod log streaming` |
| `style` | Code style | `style(format): run goimports formatter` |
| `revert` | Revert commit | `revert: undo breaking CLI flag change` |

For complete type reference, see `references/commit-types.md`.

### Scope Rules

#### Priority 1: Component from changed files

Analyze staged files to identify the primary component:

```bash
git diff --cached --name-only
```

| File pattern | Scope | Example commit |
| ------------ | ----- | -------------- |
| `pkg/cmd/pipeline/*` | `pipeline` | `feat(pipeline): add start --dry-run flag` |
| `pkg/cmd/task/*` | `task` | `fix(task): resolve list filtering` |
| `pkg/cmd/pipelinerun/*` | `pipelinerun` | `feat(pipelinerun): add cancel subcommand` |
| `pkg/cmd/taskrun/*` | `taskrun` | `fix(taskrun): correct log streaming` |
| `pkg/actions/*` | `actions` | `refactor(actions): simplify CRUD` |
| `pkg/formatted/*` | `formatted` | `feat(formatted): add JSON output` |
| `pkg/log/*` | `log` | `fix(log): handle container restart` |
| `pkg/pods/*` | `pods` | `fix(pods): resolve container lookup` |
| `pkg/cli/*` | `cli` | `refactor(cli): extract client setup` |
| `pkg/flags/*` | `flags` | `feat(flags): add --output flag` |
| `pkg/plugins/*` | `plugins` | `feat(plugins): add discovery logic` |
| `pkg/version/*` | `version` | `fix(version): correct server detection` |
| `docs/*` | `docs` or filename | `docs(README): update steps` |
| `test/*` | component being tested | `test(pipeline): add E2E tests` |
| `cmd/*` | command name | `feat(tkn): add new subcommand` |
| Root files | filename | `chore(Makefile): add target` |
| `AGENTS.md`, `CLAUDE.md` | `docs` | `docs(AGENTS.md): update conventions` |

#### Priority 2: GitHub issue number (optional)

If the work is tracked in a GitHub issue and the user provides one, it can be used as the scope:

```text
# Branch: fix-123-log-race
# Scope: #123 or component name
# Result: fix(log): resolve streaming race condition

Fixes #123
```

Add `Fixes #NNN` or `Closes #NNN` in the commit body (not the scope) — this is the standard GitHub convention for auto-closing issues.

#### Priority 3: Ask user

If changed files span multiple components or scope is unclear, ask the user which component is the primary focus.

## Line Length Requirements

### Subject Line

- **Target**: 50 characters maximum
- **Hard limit**: 72 characters
- **Format**: `type(scope): description` counts toward limit
- **Tips**: Use present tense, no period at end

```text
# Good (40 chars)
feat(pipeline): add start --dry-run

# Too long - exceeds 72 char hard limit
feat(pipeline): add comprehensive dry-run support with preview output for pipeline start command
```

### Body

- **Wrap at 72 characters per line**
- **Blank line** required between subject and body
- **Content**: Explain why, not what (code shows what)
- **Format**: Wrap manually or use heredoc in git commit

```text
feat(pipeline): add start --dry-run

Add dry-run flag to pipeline start command that previews the
PipelineRun YAML without creating it. Useful for validating
parameters and workspace bindings before submission.

Signed-off-by: Developer Name <developer@example.com>
Assisted-by: <Model name> (via <Tool>)
```

## Required Trailers

### Signed-off-by

**Always include**: `Signed-off-by: <name> <email>`

This certifies the Developer Certificate of Origin (DCO) — required by tektoncd upstream.

**Detection priority order**:

1. Environment variables: `$GIT_AUTHOR_NAME` and `$GIT_AUTHOR_EMAIL`
2. Git config: `git config user.name` and `git config user.email`
3. If neither configured, ask user to provide details

```bash
echo "$GIT_AUTHOR_NAME <$GIT_AUTHOR_EMAIL>"
git config user.name
git config user.email
```

If neither is configured, ask the user to provide their name and email.

### Assisted-by

**Always include**: `Assisted-by: <model-name> (via <Tool>)`

**Format examples**:

```text
Assisted-by: <Model name> (via <Tool>)
```

Use the actual model name (Claude Sonnet 4.5, Claude Opus 4.5, etc.).

## User Confirmation Requirement

**CRITICAL RULE**: Always ask for user confirmation before executing `git commit`.

### Confirmation Workflow

1. **Generate** the commit message following all rules above
2. **Display** the complete message to the user with separator
3. **Ask**: "Should I commit with this message? (y/n)"
4. **Wait** for user response
5. **Commit** only if user confirms (yes/y/affirmative)

### Example Interaction

```text
Generated commit message:
---
feat(pipeline): add JSON output support

Add --output=json flag to pipeline list and describe commands.
Uses the existing formatted package JSON helpers.

Signed-off-by: Developer Name <developer@example.com>
Assisted-by: <Model name> (via <Tool>)
---

Should I commit with this message? (y/n)
```

Wait for user response before proceeding.

## Commit Execution

Use heredoc format for proper multi-line handling:

```bash
git commit -m "$(cat <<'EOF'
feat(pipeline): add JSON output support

Add --output=json flag to pipeline list and describe commands.

Signed-off-by: Developer Name <developer@example.com>
Assisted-by: <Model name> (via <Tool>)
EOF
)"
```

**Never use**:

- `--no-verify` (skips pre-commit hooks)
- `--no-gpg-sign` (skips signing)
- `--amend` (unless explicitly requested and safe)

## Complete Examples

### Feature with component scope

```text
feat(pipeline): add start --dry-run

Add dry-run flag to pipeline start command that previews the
PipelineRun YAML without creating it on the cluster.

Signed-off-by: Developer Name <developer@example.com>
Assisted-by: <Model name> (via <Tool>)
```

### Bug fix closing a GitHub issue

```text
fix(log): resolve concurrent streaming panic

Prevent nil pointer dereference when multiple containers stream
logs simultaneously during a TaskRun.

Fixes #789

Signed-off-by: Developer Name <developer@example.com>
Assisted-by: <Model name> (via <Tool>)
```

### Documentation update

```text
docs(AGENTS.md): update architecture conventions

Add command tree structure and clarify golden-file testing
rules for contributors.

Signed-off-by: Developer Name <developer@example.com>
Assisted-by: <Model name> (via <Tool>)
```

### Breaking change

```text
feat(cli)!: remove deprecated --namespace flag

Remove the deprecated short-form --namespace flag from all
commands. Use -n instead. Deprecated since v0.30.

BREAKING CHANGE: Removed --namespace long flag from all
commands. Use -n or --ns instead.

Signed-off-by: Developer Name <developer@example.com>
Assisted-by: <Model name> (via <Tool>)
```

## Validation Rules

Every commit message must pass these checks:

1. **Subject line format**: Must match `type(scope): description` (conventional commits)
2. **Subject line length**: Target 50 chars, hard limit 72 chars
3. **No trailing punctuation**: Subject must not end with `.` `!` `?` `,` `:` `;` (exception: `!` for breaking changes like `feat!:`)
4. **No trailing whitespace**: On any line
5. **Blank line after subject**: Required before body text
6. **Body line length**: Wrap at 72 characters per line
7. **Valid commit type**: Must be one of: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `build`, `ci`, `perf`, `revert`
8. **Signed-off-by trailer**: Required — `Signed-off-by: Name <email>`
9. **Assisted-by trailer**: Required when AI assists — `Assisted-by: <Model name>`
10. **Trailer spacing**: Blank line before trailers, no blank lines between them, no trailing blank lines

## Format standards

Follow [Tekton commit message standards](https://github.com/tektoncd/community/blob/master/standards.md#commit-messages):

- Conventional commit format (`type(scope): description`)
- Subject line: target 50 characters, hard limit 72
- Body lines wrapped at 72 characters
- Required `Signed-off-by` trailer (DCO)
- No trailing punctuation on the subject

## Auto-Detection Summary

When generating commit messages:

1. Run `git status` (without -uall flag)
2. Run `git diff` for staged and unstaged changes
3. Identify primary component from staged file paths
4. If scope unclear, ask user
5. If user mentions a GitHub issue number, add `Fixes #NNN` to body
6. Analyze staged files to determine commit type
7. Generate appropriate scope and description
8. Detect author info from environment variables or git config
9. Ensure subject line is ≤50 characters (max 72)
10. Wrap body text at 72 characters per line
11. Add required trailers (Signed-off-by and Assisted-by)
12. Format according to conventional commits standard
13. **Display message and ask for user confirmation**
14. Only commit after receiving confirmation

## Additional Resources

For detailed information:

- **`references/commit-types.md`** - Complete commit type reference with descriptions
- **`references/trailer-detection.md`** - Author detection logic and priority order
