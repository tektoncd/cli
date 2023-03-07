## tkn hub check-upgrade

Check for upgrades of resources if present

### Usage

```
tkn hub check-upgrade
```

### Synopsis

Check for upgrades of resources if present

### Options

```
  -c, --context string      Name of the kubeconfig context to use (default: kubectl config current-context)
  -h, --help                help for check-upgrade
  -k, --kubeconfig string   Kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    Namespace to use (default: from $KUBECONFIG)
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL (default 'https://api.hub.tekton.dev' for 'tekton' type; default 'https://artifacthub.io' for 'artifact' type).
                            URL can also be defined in a file '$HOME/.tekton/hub-config' with a variable 'TEKTON_HUB_API_SERVER'/'ARTIFACT_HUB_API_SERVER'.
      --type string         The type of Hub from where to pull the resource. Either 'artifact' or 'tekton' (default "tekton")
```

### SEE ALSO

* [tkn hub](tkn_hub.md)	 - Interact with tekton hub
* [tkn hub check-upgrade task](tkn_hub_check-upgrade_task.md)	 - Check updates for Task installed via Hub CLI

