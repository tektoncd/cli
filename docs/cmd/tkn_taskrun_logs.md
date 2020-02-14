## tkn taskrun logs

Show taskruns logs

### Usage

```
tkn taskrun logs
```

### Synopsis

Show taskruns logs

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
  -h, --help           help for logs
  -L, --last           show logs for last taskrun
      --limit int      lists number of taskruns (default 5)
  -s, --step strings   show logs for mentioned steps only
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --nocolour            disable colouring (default: false)
```

### SEE ALSO

* [tkn taskrun](tkn_taskrun.md)	 - Manage taskruns

