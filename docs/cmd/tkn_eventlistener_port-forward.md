## tkn eventlistener port-forward

Port forwards an EventListener in a namespace

***Aliases**: pf*

### Usage

```
tkn eventlistener port-forward
```

### Synopsis

Port forwards an EventListener in a namespace

### Examples

Forward port '9001' from EventListener 'foo' in namespace 'bar' to localhost port '8001':

	tkn eventlistener port-forward foo -n bar -p 8001:9001

or

	tkn el pf foo -n bar -p 8001:9001


### Options

```
  -h, --help          help for port-forward
  -p, --port string   port to forward, format is [LOCAL_PORT:]REMOTE_PORT (default "8080")
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

