## tkn taskrun cancel

Cancel a TaskRun in a namespace

### Usage

```
tkn taskrun cancel
```

### Synopsis

Cancel a TaskRun in a namespace

### Examples

Cancel the TaskRun named 'foo' from namespace 'bar':

    tkn taskrun cancel foo -n bar

Cancel a TaskRun and print the result as JSON:

    tkn taskrun cancel foo -n bar -o json

Cancel a TaskRun and print the result as YAML:

    tkn taskrun cancel foo -n bar -o yaml


### Options

```
  -h, --help            help for cancel
  -o, --output string   Output format. One of: json|yaml
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

