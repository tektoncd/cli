# tkn Exit Codes

`tkn` uses a consistent set of exit codes so that scripts and CI systems can
detect success or failure without parsing command output.

## Exit Code Table

| Code | Constant       | Meaning                                    |
|------|----------------|--------------------------------------------|
| `0`  | `Success`      | The command completed successfully.        |
| `1`  | `GeneralError` | Unclassified error or command failure.     |
| `2`  | `NotFound`     | The requested resource does not exist.     |
| `3`  | `InvalidInput` | Invalid flag, parameter, or input value.   |
| `4`  | `Timeout`      | The operation exceeded its deadline.       |
| `5`  | `Unauthorized` | The request was unauthorized or forbidden. |

Exit code `127` is reserved for plugin execution failures.

## Exit Code `2` – Resource Not Found

`tkn` returns `2` whenever a Kubernetes API call returns an HTTP 404 (Not
Found). For example:

```bash
tkn taskrun describe my-missing-run -n default
# → Error: taskruns.tekton.dev "my-missing-run" not found
echo $?   # 2
```

## Exit Code `5` – Unauthorized / Forbidden

`tkn` returns `5` when the server responds with HTTP 401 or 403:

```bash
tkn pipeline list -n restricted-ns
# → Error: pipelines.tekton.dev is forbidden: ...
echo $?   # 5
```

## Structured Errors with `--output json`

When `--output json` is passed to any command that supports it, errors are
written to **stderr** as a JSON object instead of a plain-text message:

```bash
tkn pipelinerun describe missing-run --output json 2>err.json
cat err.json
# {"error":"pipelineruns.tekton.dev \"missing-run\" not found","code":2}
echo $?   # 2
```

This allows programmatic consumers to parse both the error message and the
category code without inspecting the human-readable output.

## `--exit-with-error` and PipelineRun logs

`tkn pipelinerun logs --exit-with-error` exits with the PipelineRun's Unix
status after streaming logs:

| PipelineRun state          | Exit code |
|----------------------------|-----------|
| Succeeded                  | `0`       |
| Failed                     | `1`       |
| No conditions yet          | `1`       |

> **Note:** The "no conditions" case returns `1` (general error) rather than
> `2` (not found) because the PipelineRun object exists — it simply has not
> been evaluated yet.

## Using Exit Codes in Scripts

```bash
tkn task describe my-task -n default
case $? in
  0) echo "Found" ;;
  2) echo "Task does not exist" ;;
  5) echo "Permission denied" ;;
  *) echo "Unexpected error" ;;
esac
```
