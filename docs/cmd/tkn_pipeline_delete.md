## tkn pipeline delete

Delete Pipelines in a namespace

***Aliases**: rm*

### Usage

```
tkn pipeline delete
```

### Synopsis

Delete Pipelines in a namespace

### Examples

Delete Pipelines with names 'foo' and 'bar' in namespace 'quux'

    tkn pipeline delete foo bar -n quux

or

    tkn p rm foo bar -n quux

Delete a Pipeline and print the result as JSON:

    tkn pipeline delete foo -n quux -o json

Delete a Pipeline and print the result as YAML:

    tkn pipeline delete foo -n quux -o yaml

Using -o json or -o yaml skips the confirmation prompt.


### Options

```
      --all             Delete all Pipelines in a namespace (default: false)
  -f, --force           Whether to force deletion (default: false)
  -h, --help            help for delete
  -o, --output string   Output format. One of: json|yaml. Skips the confirmation prompt
      --prs             Whether to delete Pipeline(s) and related resources (PipelineRuns) (default: false)
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --no-color            disable coloring (default: false)
```

### SEE ALSO

* [tkn pipeline](tkn_pipeline.md)	 - Manage pipelines

