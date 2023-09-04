## tkn hub upgrade task

Upgrade a Task by its name

### Usage

```
tkn hub upgrade task
```

### Synopsis

Upgrade a Task by its name

### Examples


Upgrade a Task of name 'gvr':

    tkn hub upgrade task gvr

or

Upgrade a Task of name 'gvr' to version '0.3':

    tkn hub upgrade task gvr --to 0.3


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

* [tkn hub upgrade](tkn_hub_upgrade.md)	 - Upgrade an installed resource

