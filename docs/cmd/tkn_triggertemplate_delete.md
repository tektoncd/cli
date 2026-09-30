## tkn triggertemplate delete

Delete TriggerTemplates in a namespace

***Aliases**: rm*

### Usage

```
tkn triggertemplate delete
```

### Synopsis

Delete TriggerTemplates in a namespace

### Examples

Delete TriggerTemplates with names 'foo' and 'bar' in namespace 'quux'

    tkn triggertemplate delete foo bar -n quux

or

    tkn tt rm foo bar -n quux

Delete a TriggerTemplate and print the result as JSON:

    tkn triggertemplate delete foo -n quux -o json

Delete a TriggerTemplate and print the result as YAML:

    tkn triggertemplate delete foo -n quux -o yaml

Using -o json or -o yaml skips the confirmation prompt.


### Options

```
      --all             Delete all TriggerTemplates in a namespace (default: false)
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

* [tkn triggertemplate](tkn_triggertemplate.md)	 - Manage TriggerTemplates

