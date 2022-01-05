## tkn hub install

Install a resource from a catalog by its kind, name and version

### Usage

```
tkn hub install
```

### Synopsis

Install a resource from a catalog by its kind, name and version

### Options

```
  -c, --context string      Name of the kubeconfig context to use (default: kubectl config current-context)
      --from string         Name of Catalog to which resource belongs. (default "tekton")
  -h, --help                help for install
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
* [tkn hub install task](tkn_hub_install_task.md)	 - Install Task from a catalog by its name and version

