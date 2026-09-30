## tkn customrun delete

Delete CustomRuns in a namespace

***Aliases**: rm*

### Usage

```
tkn customrun delete
```

### Synopsis

Delete CustomRuns in a namespace

### Examples

Delete CustomRun with name 'foo' in namespace 'bar':

    tkn customrun delete foo -n bar

or

    tkn cr rm foo -n bar

Delete a CustomRun and print the result as JSON:

    tkn customrun delete foo -n bar -o json

Delete a CustomRun and print the result as YAML:

    tkn customrun delete foo -n bar -o yaml


### Options

```
  -h, --help            help for delete
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

* [tkn customrun](tkn_customrun.md)	 - Manage CustomRuns

