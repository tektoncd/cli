# tkn CLI Agent Context

This file describes `tkn` conventions, output formats, and behavioral notes
for automated and agent-driven workflows. For contributing to tkn itself,
see `AGENTS.md`.

---

## Resource Types

Tekton has two layers:

- Definitions (`pipeline`, `task`): reusable templates
- Runs (`pipelinerun`, `taskrun`): execution instances

Supported verbs per resource type:

| Resource | Verbs |
|---|---|
| `pipeline`, `task` | `list`, `describe`, `start`, `logs`, `delete`, `export` |
| `pipelinerun`, `taskrun` | `list`, `describe`, `logs`, `cancel`, `delete`, `export` |

---

## Common Workflows

List pipeline runs:

```bash
tkn pipelinerun list --output json
```

Inspect run status:

```bash
tkn pipelinerun describe <run-name> --output json
tkn taskrun describe <run-name> --output json
```

Get logs for the most recent run:

```bash
tkn pipelinerun logs --last
tkn taskrun logs --last
```

Start a pipeline non-interactively:

```bash
tkn pipeline start <pipeline-name> \
  --param <key>=<value> \
  --workspace name=<ws-name>,claimName=<pvc-name> \
  --use-param-defaults \
  --skip-optional-workspace \
  --showlog
```

Re-run the most recent pipeline run:

```bash
tkn pipeline start <pipeline-name> --last --use-param-defaults
```

Delete a run:

```bash
tkn pipelinerun delete <run-name> --force
tkn taskrun delete <run-name> --force
```

---

## Output Formats

| Command | JSON output | Notes |
|---|---|---|
| `list` | Yes | Supported on all resources |
| `describe` | Yes | Supported on all resources |
| `logs` | No | Plain text to stdout only |
| `start` | No | Prints run name to stdout on success |
| `delete` | No | Requires `--force` to skip confirmation |
| `export` | No | Outputs YAML (Kubernetes manifest) |

`--output yaml` is available wherever `--output json` is supported.

---

## Non-Interactive Usage

`pipeline start` and `task start` prompt for missing parameters and workspaces
by default. In non-TTY environments, this blocks execution.

For non-interactive use, pass:

```bash
tkn pipeline start <pipeline-name> \
  --param <key>=<value> \
  --use-param-defaults \
  --skip-optional-workspace
```

`--use-param-defaults` applies schema defaults for unspecified parameters. If
a required parameter has no default and is not passed via `--param`, the
command prompts regardless.

---

## Exit Codes

`tkn pipelinerun logs` defines the following exit codes:

| Code | Meaning |
|---|---|
| 0 | PipelineRun succeeded |
| 1 | PipelineRun failed |
| 2 | No status conditions available |

Exit code behavior is only documented for `pipelinerun logs`. Other commands
do not define stable exit code semantics.

---

## Behavioral Notes

**`taskrun logs` exit code does not reflect run outcome.** A failed TaskRun
exits 0. To check TaskRun outcome, inspect `.status.conditions[0].status`:

```bash
tkn taskrun describe <run-name> --output json
```

**`delete` requires `--force` on all resources.** Without `-f`, the command
prompts for confirmation.

**`--last` targets the most recent run by creation time.** In concurrent
environments, capture the run name from `pipeline start` stdout and pass it
explicitly to subsequent commands.

**`logs` does not support structured output.** Log content goes to stdout;
errors go to stderr.

**`export` outputs YAML only.** Use `describe --output json` for structured
status inspection.

---

## Common Flags

| Flag | Commands | Purpose |
|---|---|---|
| `--output json` | `list`, `describe` | Machine-readable output |
| `--last` | `logs`, `start`, `describe` | Target most recent run |
| `--use-param-defaults` | `start` | Use schema defaults for unspecified params |
| `--skip-optional-workspace` | `start` | Skip prompts for optional workspaces |
| `--force`, `-f` | `delete` | Skip confirmation prompt |
| `--showlog` | `start` | Stream logs after starting |
| `--namespace`, `-n` | all | Override kubeconfig namespace |
| `--param`, `-p` | `start` | Pass parameter as `<key>=<value>` |
| `--workspace`, `-w` | `start` | Pass workspace binding |