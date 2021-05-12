## tkn hub check-upgrade task

Check updates for Task installed via Hub CLI

### Usage

```
tkn hub check-upgrade task
```

### Synopsis

Check updates for Task installed via Hub CLI

### Examples


Check for Upgrades of Task installed via Tekton Hub CLI:

	tkn hub check-upgrades task

The above command will check for upgrades of Tasks installed via Tekton Hub CLI
and will skip the Tasks which are not installed by Tekton Hub CLI.

NOTE: If Pipelines version is unknown it will show the latest version available
else it will show latest compatible version.


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
```

### SEE ALSO

* [tkn hub check-upgrade](tkn_hub_check-upgrade.md)	 - Check for upgrades of resources if present

