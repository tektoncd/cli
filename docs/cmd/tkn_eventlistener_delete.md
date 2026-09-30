## tkn eventlistener delete

Delete EventListeners in a namespace

***Aliases**: rm*

### Usage

```
tkn eventlistener delete
```

### Synopsis

Delete EventListeners in a namespace

### Examples

Delete EventListeners with names 'foo' and 'bar' in namespace 'bar'

    tkn eventlistener delete foo bar -n quux

or

    tkn el rm foo bar -n quux

Delete an EventListener and print the result as JSON:

    tkn eventlistener delete foo -n quux -o json

Delete an EventListener and print the result as YAML:

    tkn eventlistener delete foo -n quux -o yaml

Using -o json or -o yaml skips the confirmation prompt.


### Options

```
      --all             Delete all EventListeners in a namespace (default: false)
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

* [tkn eventlistener](tkn_eventlistener.md)	 - Manage EventListeners

