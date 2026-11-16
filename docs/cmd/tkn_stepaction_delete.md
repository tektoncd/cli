## tkn stepaction delete

Delete StepActions in a namespace

***Aliases**: rm*

### Usage

```
tkn stepaction delete
```

### Synopsis

Delete StepActions in a namespace

### Examples

Delete StepAction with name 'foo' in namespace 'bar':

    tkn stepaction delete foo -n bar

or

    tkn stepaction rm foo -n bar

Delete a StepAction and print the result as JSON:

    tkn stepaction delete foo -n bar -o json

Delete a StepAction and print the result as YAML:

    tkn stepaction delete foo -n bar -o yaml


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

* [tkn stepaction](tkn_stepaction.md)	 - Manage StepActions

