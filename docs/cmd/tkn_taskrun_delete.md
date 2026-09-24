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

or

    tkn tr rm foo bar -n quux


### Options

```
      --all                          Delete all TaskRuns in a namespace (default: false)
  -f, --force                        Whether to force deletion (default: false)
  -h, --help                         help for delete
  -i, --ignore-running               ignore running TaskRun (default true)
      --ignore-running-pipelinerun   ignore deleting taskruns of a running PipelineRun (default true)
      --keep int                     Keep n most recent number of TaskRuns
      --keep-since int               When deleting all TaskRuns keep the ones that has been completed since n minutes
  -o, --output string                Output format. Only "json" is supported
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

