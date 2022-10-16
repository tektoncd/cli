## tkn taskrun logs

Show TaskRuns logs

### Usage

```
tkn taskrun logs
```

### Synopsis

Show TaskRuns logs

### Examples


Show the logs of TaskRun named 'foo' from the namespace 'bar':

    tkn taskrun logs foo -n bar

Show the live logs of TaskRun named 'foo' from namespace 'bar':

    tkn taskrun logs -f foo -n bar

Show the logs of TaskRun named 'microservice-1' for step 'build' only from namespace 'bar':

    tkn tr logs microservice-1 -s build -n bar


### Options

```
  -a, --all            show all logs including init steps injected by tekton
  -f, --follow         stream live logs
  -F, --fzf            use fzf to select a TaskRun
  -h, --help           help for logs
  -L, --last           show logs for last TaskRun
      --limit int      lists number of TaskRuns (default 5)
      --prefix         prefix each log line with the log source (step name) (default true)
  -s, --step strings   show logs for mentioned steps only
  -t, --timestamps     show logs with timestamp
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

