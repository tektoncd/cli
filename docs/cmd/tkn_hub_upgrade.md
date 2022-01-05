## tkn hub upgrade

Upgrade an installed resource

### Usage

```
tkn hub upgrade
```

### Synopsis

Upgrade an installed resource

### Options

```
  -c, --context string      Name of the kubeconfig context to use (default: kubectl config current-context)
  -h, --help                help for upgrade
  -k, --kubeconfig string   Kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    Namespace to use (default: from $KUBECONFIG)
      --to string           Version of Resource
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL (default "https://api.hub.tekton.dev")
```

### SEE ALSO

* [tkn hub](tkn_hub.md)	 - Interact with tekton hub
* [tkn hub upgrade task](tkn_hub_upgrade_task.md)	 - Upgrade a Task by its name

