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

Delete a PipelineRun and print the result as JSON:

    tkn pipelinerun delete foo -n quux -o json

Delete a PipelineRun and print the result as YAML:

    tkn pipelinerun delete foo -n quux -o yaml

Delete all PipelineRuns in a namespace and print the result as JSON:

    tkn pipelinerun delete --all -n quux -o json

Delete all PipelineRuns in a namespace and print the result as YAML:

    tkn pipelinerun delete --all -n quux -o yaml

Delete all but the 2 most recent PipelineRuns and print the result as JSON:

    tkn pipelinerun delete --keep 2 -n quux -o json

Delete all but the 2 most recent PipelineRuns and print the result as YAML:

    tkn pipelinerun delete --keep 2 -n quux -o yaml

Delete PipelineRuns for a Pipeline and print the result as JSON:

    tkn pipelinerun delete --pipeline foo -n quux -o json

Delete PipelineRuns for a Pipeline and print the result as YAML:

    tkn pipelinerun delete --pipeline foo -n quux -o yaml

Using -o json or -o yaml skips the confirmation prompt.


### Options

```
      --all               Delete all PipelineRuns in a namespace (default: false)
  -f, --force             Whether to force deletion (default: false)
  -h, --help              help for delete
  -i, --ignore-running    ignore running PipelineRun (default true)
      --keep int          Keep n most recent number of PipelineRuns
      --keep-since int    When deleting all PipelineRuns keep the ones that has been completed since n minutes
      --label string      A selector (label query) to filter on when running with --all, supports '=', '==', and '!='
  -o, --output string     Output format. One of: json|yaml. Skips the confirmation prompt
  -p, --pipeline string   The name of a Pipeline whose PipelineRuns should be deleted (does not delete the Pipeline)
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

