## tkn hub reinstall

Reinstall a resource by its kind and name

### Usage

```
tkn hub reinstall
```

### Synopsis

Reinstall a resource by its kind and name

### Options

```
  -c, --context string      Name of the kubeconfig context to use (default: kubectl config current-context)
      --from string         Name of Catalog to which resource belongs. (default "tekton")
  -h, --help                help for reinstall
  -k, --kubeconfig string   Kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    Namespace to use (default: from $KUBECONFIG)
      --version string      Version of Resource
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL (default "https://api.hub.tekton.dev")
```

### SEE ALSO

* [tkn hub](tkn_hub.md)	 - Interact with tekton hub
* [tkn hub reinstall task](tkn_hub_reinstall_task.md)	 - Reinstall a Task by its name

