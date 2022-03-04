## tkn hub reinstall task

Reinstall a Task by its name

### Usage

```
tkn hub reinstall task
```

### Synopsis

Reinstall a Task by its name

### Examples


Reinstall a Task of name 'foo':

    tkn hub reinstall task foo

or

Reinstall a Task of name 'foo' of version '0.3' from Catalog 'Tekton':

	tkn hub reinstall task foo --version 0.3 --from tekton


### Options

```
  -h, --help   help for task
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL (default 'https://api.hub.tekton.dev').
                            URL can also be defined in a file '$HOME/.tekton/hub-config' with a variable 'HUB_API_SERVER'.
  -c, --context string      Name of the kubeconfig context to use (default: kubectl config current-context)
      --from string         Name of Catalog to which resource belongs. (default "tekton")
  -k, --kubeconfig string   Kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    Namespace to use (default: from $KUBECONFIG)
      --version string      Version of Resource
```

### SEE ALSO

* [tkn hub reinstall](tkn_hub_reinstall.md)	 - Reinstall a resource by its kind and name

