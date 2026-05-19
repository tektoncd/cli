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
      --from string         Name of Catalog to which resource belongs.
  -h, --help                help for install
  -k, --kubeconfig string   Kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    Namespace to use (default: from $KUBECONFIG)
      --version string      Version of Resource
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL.
                            For artifact type: default 'https://artifacthub.io' (env: ARTIFACT_HUB_API_SERVER)
                            For tekton type (DEPRECATED): default 'https://api.hub.tekton.dev' (env: TEKTON_HUB_API_SERVER)
                            Can also be set in '$HOME/.tekton/hub-config'.
      --type string         The type of Hub from where to pull the resource. Either 'artifact' or 'tekton' (DEPRECATED: tekton type will be removed in a future release) (default "artifact")
```

### SEE ALSO

* [tkn hub](tkn_hub.md)	 - Interact with artifacthub
* [tkn hub install task](tkn_hub_install_task.md)	 - Install Task from a catalog by its name and version

