## tkn clustertask delete

Delete ClusterTasks in a cluster

***Aliases**: rm*

### Usage

```
tkn clustertask delete
```

### Synopsis

Delete ClusterTasks in a cluster

### Examples

Delete ClusterTasks with names 'foo' and 'bar':

    tkn clustertask delete foo bar

or

    tkn ct rm foo bar


### Options

```
      --all                           Delete all ClusterTasks (default: false)
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -f, --force                         Whether to force deletion (default: false)
  -h, --help                          help for delete
  -o, --output string                 Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-as-json|jsonpath-file.
      --show-managed-fields           If true, keep the managedFields when printing objects in JSON or YAML format.
      --template string               Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].
      --trs                           Whether to delete ClusterTask(s) and related resources (TaskRuns) (default: false)
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -C, --no-color            disable coloring (default: false)
```

### SEE ALSO

* [tkn clustertask](tkn_clustertask.md)	 - Manage ClusterTasks

