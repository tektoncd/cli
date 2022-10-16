## tkn task logs

Show Task logs

### Usage

```
tkn task logs
```

### Synopsis

Show Task logs

### Examples

Interactive mode: shows logs of the selected TaskRun:

    tkn task logs -n namespace

Interactive mode: shows logs of the selected TaskRun of the given Task:

    tkn task logs task -n namespace

Show logs of given Task for last TaskRun:

    tkn task logs task -n namespace --last

Show logs for given Task and associated TaskRun:

    tkn task logs task taskrun -n namespace


### Options

```
  -a, --all          show all logs including init steps injected by tekton
  -f, --follow       stream live logs
  -h, --help         help for logs
  -L, --last         show logs for last TaskRun
      --limit int    lists number of TaskRuns (default 5)
  -t, --timestamps   show logs with timestamp
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --no-color            disable coloring (default: false)
```

### SEE ALSO

* [tkn task](tkn_task.md)	 - Manage Tasks

