# Author Detection and Trailer Generation

Complete guide for detecting author information and generating required commit trailers.

## Overview

Every commit must include two trailers:

1. **Signed-off-by**: Author's name and email
2. **Assisted-by**: AI model information (when AI assists)

This document describes the complete detection logic and trailer generation process.

## Signed-off-by Trailer

### Purpose

The `Signed-off-by` trailer indicates who created the commit and certifies that the contributor has the right to submit the code.

**Format**: `Signed-off-by: Full Name <email>`

### Detection Priority Order

Author information is detected using this priority:

#### Priority 1: Environment Variables (Highest)

Check environment variables first (common in dev containers and CI):

```bash
# Check if both variables are set
if [ -n "$GIT_AUTHOR_NAME" ] && [ -n "$GIT_AUTHOR_EMAIL" ]; then
    author_name="$GIT_AUTHOR_NAME"
    author_email="$GIT_AUTHOR_EMAIL"
fi
```

**Environment variables**:

- `$GIT_AUTHOR_NAME` - Author's full name
- `$GIT_AUTHOR_EMAIL` - Author's email address

**Example**:

```bash
export GIT_AUTHOR_NAME="Jane Developer"
export GIT_AUTHOR_EMAIL="jane.developer@example.com"
```

**Result**:

```text
Signed-off-by: Jane Developer <jane.developer@example.com>
```

**When used**:

- Development containers (devcontainers)
- CI/CD pipelines
- Containerized environments
- Explicitly set environments

#### Priority 2: Git Configuration (Fallback)

If environment variables not set, check git configuration:

```bash
# Get author name from git config
author_name=$(git config user.name)
author_email=$(git config user.email)

if [ -n "$author_name" ] && [ -n "$author_email" ]; then
    echo "Using git config: $author_name <$author_email>"
fi
```

**Git config sources** (in priority order):

1. **Repository config** (`.git/config`): `git config user.name`
2. **Global config** (`~/.gitconfig`): `git config --global user.name`
3. **System config** (`/etc/gitconfig`): `git config --system user.name`

**Example**:

```bash
# Set globally
git config --global user.name "John Developer"
git config --global user.email "john.developer@example.com"

# Or per-repository
git config user.name "John Developer"
git config user.email "john.developer@example.com"
```

**Result**:

```text
Signed-off-by: John Developer <john.developer@example.com>
```

#### Priority 3: Ask User (Last Resort)

If neither environment variables nor git config available:

**Prompt**:

```text
Git author information not configured. Please provide:

Full Name: _
Email: _
```

**Validation**:

- Name: Non-empty, contains at least first and last name
- Email: Valid email format (`user@domain.com`)

**Store for session**:
After user provides information, optionally ask:

```text
Would you like to save this information?
1. For this repository only (git config)
2. Globally for all repositories (git config --global)
3. Just for this commit (don't save)
```

### Complete Detection Logic

```bash
detect_author() {
    local author_name=""
    local author_email=""

    # Priority 1: Environment variables
    if [ -n "$GIT_AUTHOR_NAME" ] && [ -n "$GIT_AUTHOR_EMAIL" ]; then
        author_name="$GIT_AUTHOR_NAME"
        author_email="$GIT_AUTHOR_EMAIL"
        echo "Using environment variables" >&2

    # Priority 2: Git config
    elif git config user.name >/dev/null && git config user.email >/dev/null; then
        author_name=$(git config user.name)
        author_email=$(git config user.email)
        echo "Using git config" >&2

    # Priority 3: Ask user
    else
        echo "Git author information not configured." >&2
        read -p "Full Name: " author_name
        read -p "Email: " author_email

        if [ -z "$author_name" ] || [ -z "$author_email" ]; then
            echo "Error: Name and email are required" >&2
            return 1
        fi
    fi

    echo "Signed-off-by: $author_name <$author_email>"
}
```

### Common Scenarios

#### Scenario 1: DevContainer with Environment Variables

```json
{
  "containerEnv": {
    "GIT_AUTHOR_NAME": "Developer Name",
    "GIT_AUTHOR_EMAIL": "developer@example.com"
  }
}
```

**Detection**: Uses environment variables (Priority 1)
**Result**: `Signed-off-by: Developer Name <developer@example.com>`

#### Scenario 2: Local Development with Git Config

```ini
# User's ~/.gitconfig
[user]
    name = John Developer
    email = john@example.com
```

**Detection**: Uses git config (Priority 2)
**Result**: `Signed-off-by: John Developer <john@example.com>`

#### Scenario 3: Multiple Git Identities

User has different identities for different projects:

```bash
# Global config (personal)
git config --global user.name "Jane Doe"
git config --global user.email "jane@personal.com"

# Repository config (work)
cd /work/project
git config user.name "Jane Developer"
git config user.email "jane.developer@company.com"
```

**Detection**: Uses repository config (overrides global)
**Result**: `Signed-off-by: Jane Developer <jane.developer@company.com>`

## Assisted-by Trailer

### Purpose

The `Assisted-by` trailer credits AI assistance in creating the commit, following open source contribution practices.

**Format**: `Assisted-by: Model Name (via Tool Name)`

### Model Name Detection

Determine the current AI model being used:

**Available models**:

- Claude Sonnet 4.5
- Claude Opus 4.5
- (Other Claude models)

**Detection method**: Check model identifier or configuration

**Examples**:

```text
Assisted-by: Claude Sonnet 4.5 (via Cursor)
Assisted-by: Claude Opus 4.5 (via Cursor)
```

### Tool Name

Use the tool that is driving the session (e.g., `Cursor`, `Claude Code`, etc.).

### Format Rules

1. **Model name first**: Full model name (e.g., `Claude Sonnet 4.5`)
2. **Tool in parentheses**: `(via Tool Name)`
3. **Consistent casing**: Proper names capitalized
4. **No abbreviations**: Use full names

**Correct**:

```text
Assisted-by: Claude Sonnet 4.5 (via Cursor)
```

**Incorrect**:

```text
Assisted-by: claude-sonnet (via cursor)           # Wrong casing
Assisted-by: Sonnet (via C)                       # Abbreviations
Assisted-by: (via Cursor) Claude Sonnet 4.5       # Wrong order
Assisted-by: Claude Sonnet 4.5                    # Missing tool
```

### When to Include

**Always include** when:

- AI generated commit message
- AI assisted with commit message
- AI helped analyze changes
- AI suggested improvements

## Complete Trailer Generation

### Both Trailers Together

Always include both trailers in this order:

1. Signed-off-by (required)
2. Assisted-by (when AI assists)

**Format**:

```text
Signed-off-by: Full Name <email@example.com>
Assisted-by: Model Name (via Tool Name)
```

### Complete Example

```text
feat(pipeline): add start --dry-run flag

Add dry-run flag to pipeline start command that previews the
PipelineRun YAML without creating it on the cluster.

Signed-off-by: Jane Developer <jane.developer@example.com>
Assisted-by: Claude Sonnet 4.5 (via Cursor)
```

### Spacing Rules

- **Blank line before trailers**: Separate body from trailers
- **No blank lines between trailers**: Trailers are consecutive
- **No trailing blank lines**: End commit message after last trailer

**Correct**:

```text
feat(pipeline): add start handler

Implements pipeline start support.

Signed-off-by: Jane Developer <jane@example.com>
Assisted-by: Claude Sonnet 4.5 (via Cursor)
```

**Incorrect**:

```text
feat(pipeline): add start handler

Implements pipeline start support.
Signed-off-by: Jane Developer <jane@example.com>

Assisted-by: Claude Sonnet 4.5 (via Cursor)

```

(Missing blank line before trailers, extra blank lines between/after)

## Trailer Generation Function

Complete implementation:

```bash
generate_trailers() {
    local author_name=""
    local author_email=""
    local model_name="Claude Sonnet 4.5"  # Detect actual model
    local tool_name="Cursor"

    # Detect author (Priority 1: env vars)
    if [ -n "$GIT_AUTHOR_NAME" ] && [ -n "$GIT_AUTHOR_EMAIL" ]; then
        author_name="$GIT_AUTHOR_NAME"
        author_email="$GIT_AUTHOR_EMAIL"

    # Priority 2: git config
    elif git config user.name >/dev/null && git config user.email >/dev/null; then
        author_name=$(git config user.name)
        author_email=$(git config user.email)

    # Priority 3: ask user
    else
        read -p "Full Name: " author_name
        read -p "Email: " author_email
    fi

    # Generate trailers
    echo ""  # Blank line before trailers
    echo "Signed-off-by: $author_name <$author_email>"
    echo "Assisted-by: $model_name (via $tool_name)"
}
```

## Troubleshooting

### Issue: "user.name" not set

**Error**: `fatal: unable to auto-detect email address`

**Fix**:

```bash
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"
```

### Issue: Wrong author in devcontainer

**Cause**: Environment variables override git config

**Check**:

```bash
echo "$GIT_AUTHOR_NAME <$GIT_AUTHOR_EMAIL>"
```

**Fix**: Set correct environment variables in `.devcontainer/devcontainer.json`

### Issue: Trailers in wrong order

**Wrong**:

```text
Assisted-by: Claude Sonnet 4.5 (via Cursor)
Signed-off-by: Jane Developer <jane@example.com>
```

**Correct**:

```text
Signed-off-by: Jane Developer <jane@example.com>
Assisted-by: Claude Sonnet 4.5 (via Cursor)
```

**Fix**: Always put Signed-off-by first, Assisted-by second

## Summary

Trailer detection priority:

1. Environment variables (`$GIT_AUTHOR_NAME`, `$GIT_AUTHOR_EMAIL`)
2. Git configuration (`git config user.name`, `git config user.email`)
3. Ask user (last resort)

Required trailers (in order):

1. `Signed-off-by: Name <email>`
2. `Assisted-by: Model Name (via Tool Name)`
