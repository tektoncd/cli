## tkn hub downgrade task

Downgrade an installed Task by its name to a lower version

### Usage

```
tkn hub downgrade task
```

### Synopsis

Downgrade an installed Task by its name to a lower version

### Examples


Downgrade a Task of name 'foo' to previous version:

    tkn hub downgrade task foo

or

Downgrade a Task of name 'foo' to version '0.3':

    tkn hub downgrade task foo --to 0.3


### Options

```
  -h, --help   help for task
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL (default 'https://api.hub.tekton.dev' for 'tekton' type; default 'https://artifacthub.io' for 'artifact' type).
                            URL can also be defined in a file '$HOME/.tekton/hub-config' with a variable 'TEKTON_HUB_API_SERVER'/'ARTIFACT_HUB_API_SERVER'.
  -c, --context string      Name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   Kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    Namespace to use (default: from $KUBECONFIG)
      --to string           Version of Resource
      --type string         The type of Hub from where to pull the resource. Either 'artifact' or 'tekton' (default "tekton")
```

### SEE ALSO

* [tkn hub downgrade](tkn_hub_downgrade.md)	 - Downgrade an installed resource

