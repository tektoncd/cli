## tkn hub downgrade

Downgrade an installed resource

### Usage

```
tkn hub downgrade
```

### Synopsis

Downgrade an installed resource

### Options

```
  -c, --context string      Name of the kubeconfig context to use (default: kubectl config current-context)
  -h, --help                help for downgrade
  -k, --kubeconfig string   Kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    Namespace to use (default: from $KUBECONFIG)
      --to string           Version of Resource
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL (default 'https://api.hub.tekton.dev').
                            URL can also be defined in a file '$HOME/.tekton/hub-config' with a variable 'HUB_API_SERVER'.
```

### SEE ALSO

* [tkn hub](tkn_hub.md)	 - Interact with tekton hub
* [tkn hub downgrade task](tkn_hub_downgrade_task.md)	 - Downgrade an installed Task by its name to a lower version

