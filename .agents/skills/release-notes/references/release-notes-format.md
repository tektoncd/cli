# Release Notes Format Reference

## Entry Format

Each release note entry MUST follow this exact format. The Link line MUST be indented with two spaces so it renders as a nested sub-bullet:

```markdown
* **Bold title:** One-sentence description of the change.
  * Link: <PR_OR_COMMIT_URL>
```

### Rules

- The first bullet MUST start with `*` (no indent) with a bold title followed by a colon and a description.
- The Link line MUST start with `* Link:` (two-space indent) with the PR or commit URL.
- Do NOT add a Contributors section.

## Header Template

```markdown
# Tekton CLI {tag}

Tekton CLI {tag} has been released 🎉
```

## Section Headers

Use exactly these section headers (skip empty ones):

```markdown
## ✨ Major changes and Features
## 🐛 Bug Fixes
## 📚 Documentation Updates
## ⚙️ Chores
```

## Installation Section Template

```markdown
## Installation

Install `tkn` using one of the following methods:

### Homebrew (macOS/Linux)

\`\`\`shell
brew install tektoncd-cli
\`\`\`

### Release binary

Download the binary for your platform from the [GitHub release page](https://github.com/tektoncd/cli/releases/tag/{tag}).

### From source

\`\`\`shell
go install github.com/tektoncd/cli/cmd/tkn@{tag}
\`\`\`

### Documentation

https://github.com/tektoncd/cli/tree/{tag}/docs
```

## Complete Example

```markdown
# Tekton CLI v0.45.0

Tekton CLI v0.45.0 has been released 🎉

## ✨ Major changes and Features

* **Add pipeline start --dry-run flag:** Preview PipelineRun YAML without creating it on the cluster.
  * Link: https://github.com/tektoncd/cli/pull/1234
* **Support JSON output for task describe:** Add --output=json flag to task describe command.
  * Link: https://github.com/tektoncd/cli/pull/1230

## 🐛 Bug Fixes

* **Fix log streaming race condition:** Corrected concurrent container log streaming that caused nil pointer panics.
  * Link: https://github.com/tektoncd/cli/pull/1220

## ⚙️ Chores

* **Bump go.opentelemetry.io/otel from 1.28.0 to 1.29.0:** Updated OpenTelemetry dependency to latest version.
  * Link: https://github.com/tektoncd/cli/pull/1215

## Installation

Install `tkn` using one of the following methods:

### Homebrew (macOS/Linux)

\`\`\`shell
brew install tektoncd-cli
\`\`\`

### Release binary

Download the binary for your platform from the [GitHub release page](https://github.com/tektoncd/cli/releases/tag/v0.45.0).

### From source

\`\`\`shell
go install github.com/tektoncd/cli/cmd/tkn@v0.45.0
\`\`\`

### Documentation

https://github.com/tektoncd/cli/tree/v0.45.0/docs

## What's Changed
<!-- GitHub auto-generated changelog goes here -->
```
