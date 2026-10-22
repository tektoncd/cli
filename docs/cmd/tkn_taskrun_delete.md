## tkn taskrun delete

Delete TaskRuns in a namespace

***Aliases**: rm*

### Usage

```
tkn taskrun delete
```

### Synopsis

Delete TaskRuns in a namespace

### Examples

Delete TaskRuns with names 'foo' and 'bar' in namespace 'quux':

    tkn taskrun delete foo bar -n quux

Delete TaskRuns and print JSON:

    tkn taskrun delete foo bar -n quux -o json

Delete all TaskRuns and print JSON:

    tkn taskrun delete --all -n quux -o json

Delete TaskRuns associated with a Task:

    tkn taskrun delete --task my-task -n quux -o json

or

    tkn tr rm foo bar -n quux

Deleting a TaskRun owned by a running PipelineRun requires --force when using structured output.

### Options

```
      --all                          Delete all TaskRuns in a namespace (default: false)
  -f, --force                        Whether to force deletion (default: false)
  -h, --help                         help for delete
  -i, --ignore-running               ignore running TaskRun (default true)
      --ignore-running-pipelinerun   ignore deleting taskruns of a running PipelineRun (default true)
      --keep int                     Keep n most recent number of TaskRuns
      --keep-since int               When deleting all TaskRuns keep the ones that has been completed since n minutes
  -o, --output string                Output format. One of: json|yaml. Skips the confirmation prompt
  -t, --task string                  The name of a Task whose TaskRuns should be deleted (does not delete the task)
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --no-color            disable coloring (default: false)
```

### SEE ALSO

* [tkn taskrun](tkn_taskrun.md)	 - Manage TaskRuns

