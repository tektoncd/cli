## tkn clustertask logs

Show ClusterTask logs

### Usage

```
tkn clustertask logs
```

### Synopsis

Show ClusterTask logs

### Examples

Interactive mode: shows logs of the selected TaskRun:

    tkn clustertask logs -n namespace

Interactive mode: shows logs of the selected TaskRun of the given ClusterTask:

    tkn clustertask logs clustertask -n namespace

Show logs of given ClusterTask for last TaskRun:

    tkn clustertask logs clustertask -n namespace --last

Show logs for given ClusterTask and associated TaskRun:

    tkn clustertask logs clustertask taskrun -n namespace


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
  -C, --no-color            disable coloring (default: false)
```

### SEE ALSO

* [tkn clustertask](tkn_clustertask.md)	 - Manage ClusterTasks

