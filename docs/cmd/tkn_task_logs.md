## tkn task logs

Show task logs

### Usage

```
tkn task logs
```

### Synopsis

Show task logs

### Examples


  # interactive mode: shows logs of the selected taskrun
    tkn task logs -n namespace

  # interactive mode: shows logs of the selected taskrun of the given task
    tkn task logs task -n namespace

  # show logs of given task for last taskrun
    tkn task logs task -n namespace --last

  # show logs for given task and taskrun
    tkn task logs task taskrun -n namespace

   

### Options

```
  -a, --all         show all logs including init steps injected by tekton
  -f, --follow      stream live logs
  -h, --help        help for logs
  -L, --last        show logs for last taskrun
      --limit int   lists number of taskruns (default 5)
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --nocolour            disable colouring (default: false)
```

### SEE ALSO

* [tkn task](tkn_task.md)	 - Manage tasks

