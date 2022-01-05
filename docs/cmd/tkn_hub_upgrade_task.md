## tkn hub upgrade task

Upgrade a Task by its name

### Usage

```
tkn hub upgrade task
```

### Synopsis

Upgrade a Task by its name

### Examples


Upgrade a Task of name 'foo':

    tkn hub upgrade task foo

or

Upgrade a Task of name 'foo' to version '0.3':

    tkn hub upgrade task foo --to 0.3


### Options

```
  -h, --help   help for task
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL (default "https://api.hub.tekton.dev")
  -c, --context string      Name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   Kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    Namespace to use (default: from $KUBECONFIG)
      --to string           Version of Resource
```

### SEE ALSO

* [tkn hub upgrade](tkn_hub_upgrade.md)	 - Upgrade an installed resource

