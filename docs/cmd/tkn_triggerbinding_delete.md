## tkn triggerbinding delete

Delete TriggerBindings in a namespace

***Aliases**: rm*

### Usage

```
tkn triggerbinding delete
```

### Synopsis

Delete TriggerBindings in a namespace

### Examples

Delete TriggerBindings with names 'foo' and 'bar' in namespace 'quux'

    tkn triggerbinding delete foo bar -n quux

or

    tkn tb rm foo bar -n quux

Delete a TriggerBinding and print the result as JSON:

    tkn triggerbinding delete foo -n quux -o json

Delete a TriggerBinding and print the result as YAML:

    tkn triggerbinding delete foo -n quux -o yaml

Using -o json or -o yaml skips the confirmation prompt.


### Options

```
      --all             Delete all TriggerBindings in a namespace (default: false)
  -f, --force           Whether to force deletion (default: false)
  -h, --help            help for delete
  -o, --output string   Output format. One of: json|yaml. Skips the confirmation prompt
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --no-color            disable coloring (default: false)
```

### SEE ALSO

* [tkn triggerbinding](tkn_triggerbinding.md)	 - Manage TriggerBindings

