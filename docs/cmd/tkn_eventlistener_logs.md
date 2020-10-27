## tkn eventlistener logs

Show EventListener logs

### Usage

```
tkn eventlistener logs
```

### Synopsis

Show EventListener logs

### Examples


Show logs of EventListener pods:

    tkn eventlistener logs eventlistenerName

Show 2 lines of most recent logs from all EventListener pods:

    tkn eventlistener logs eventListenerName -t 2

### Options

```
  -h, --help       help for logs
  -t, --tail int   Number of most recent log lines to show. Specify -1 for all logs from each pod. (default 10)
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --no-color            disable coloring (default: false)
```

### SEE ALSO

* [tkn eventlistener](tkn_eventlistener.md)	 - Manage EventListeners

