## tkn taskrun restart

Restart a TaskRun in a namespace

### Usage

```
tkn taskrun restart
```

### Synopsis

Restart a TaskRun in a namespace

### Examples

Restart a TaskRun named foo in namespace bar:

	tkn taskrun restart foo -n bar


### Options

```
  -h, --help                 help for restart
      --prefix-name string   Specify a prefix for the TaskRun name (must be lowercase alphanumeric characters)
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --no-color            disable coloring (default: false)
```

### SEE ALSO

* [tkn taskrun](tkn_taskrun.md)	 - Manage TaskRuns

