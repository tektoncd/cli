## tkn clustertriggerbinding delete

Delete ClusterTriggerBindings

***Aliases**: rm*

### Usage

```
tkn clustertriggerbinding delete
```

### Synopsis

Delete ClusterTriggerBindings

### Examples

Delete ClusterTriggerBindings with names 'foo' and 'bar'

    tkn clustertriggerbinding delete foo bar

or

    tkn ctb rm foo bar

Delete a ClusterTriggerBinding and print the result as JSON:

    tkn clustertriggerbinding delete foo -o json

Delete a ClusterTriggerBinding and print the result as YAML:

    tkn clustertriggerbinding delete foo -o yaml

Using -o json or -o yaml skips the confirmation prompt.


### Options

```
      --all             Delete all ClusterTriggerBindings (default: false)
  -f, --force           Whether to force deletion (default: false)
  -h, --help            help for delete
  -o, --output string   Output format. One of: json|yaml. Skips the confirmation prompt
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -C, --no-color            disable coloring (default: false)
```

### SEE ALSO

* [tkn clustertriggerbinding](tkn_clustertriggerbinding.md)	 - Manage ClusterTriggerBindings

