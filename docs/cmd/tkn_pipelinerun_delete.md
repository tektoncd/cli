## tkn pipelinerun delete

Delete PipelineRuns in a namespace

***Aliases**: rm*

### Usage

```
tkn pipelinerun delete
```

### Synopsis

Delete PipelineRuns in a namespace

### Examples

Delete PipelineRuns with names 'foo' and 'bar' in namespace 'quux':

    tkn pipelinerun delete foo bar -n quux

or

    tkn pr rm foo bar -n quux


### Options

```
      --all                           Delete all PipelineRuns in a namespace (default: false)
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -f, --force                         Whether to force deletion (default: false)
  -h, --help                          help for delete
      --keep int                      Keep n most recent number of PipelineRuns
      --keep-since int                When deleting all PipelineRuns keep the ones that has been completed since n minutes
      --label string                  A selector (label query) to filter on when running with --all, supports '=', '==', and '!='
  -o, --output string                 Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-as-json|jsonpath-file.
  -p, --pipeline string               The name of a Pipeline whose PipelineRuns should be deleted (does not delete the Pipeline)
      --show-managed-fields           If true, keep the managedFields when printing objects in JSON or YAML format.
      --template string               Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --no-color            disable coloring (default: false)
```

### SEE ALSO

* [tkn pipelinerun](tkn_pipelinerun.md)	 - Manage PipelineRuns

