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
| `pipeline` | `list`, `describe`, `start`, `logs`, `delete` |
| `task` | `list`, `describe`, `start`, `logs`, `delete` |
| `pipelinerun`, `taskrun` | `list`, `describe`, `logs`, `cancel`, `delete` |

Other resource types (`customrun`, `eventlistener`, `triggerbinding`,
`triggertemplate`, `clustertriggerbinding`, `bundle`, and `hub`) support
subsets of `list`, `describe`, and `delete`.

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
| `start` | Yes | Supported by `pipeline start` and `task start` |
| `logs` | No | Plain text to stdout only |
| `delete` | No | Requires `--force` to skip confirmation |
| `export` | No | Outputs YAML instead of JSON |

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

By default, `tkn pipelinerun logs` exits 0 after streaming logs, regardless
of run outcome.

With `--exit-with-pipelinerun-error` (`-E`), the exit code reflects run status:

| Code | Meaning |
|---|---|
| 0 | PipelineRun succeeded |
| 1 | PipelineRun failed |
| 2 | No status conditions available |

`-E` is also available on `pipeline start` when used with `--showlog`.

`taskrun logs` does not support this flag — it always exits 0.

---

## Behavioral Notes

**`taskrun logs` has no equivalent of `-E`.** To check TaskRun outcome,
inspect `.status.conditions[0].status`:

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
| `--output json` | `list`, `describe`, `pipeline start` | Machine-readable output |
| `--last` | `logs`, `start`, `pipelinerun describe`, `taskrun describe` | Target the most recent run |
| `--use-param-defaults` | `start` | Use schema defaults for unspecified parameters |
| `--skip-optional-workspace` | `start` | Skip prompts for optional workspaces |
| `--force`, `-f` | `delete` | Skip confirmation prompt |
| `--showlog` | `start` | Stream logs after starting |
| `--exit-with-pipelinerun-error`, `-E` | `pipelinerun logs`, `pipeline start` | Exit with PipelineRun status (0=success, 1=failed, 2=unknown) |
| `--namespace`, `-n` | all | Override the kubeconfig namespace |
| `--param`, `-p` | `start` | Pass parameter as `<key>=<value>` |
| `--workspace`, `-w` | `start` | Pass workspace binding |