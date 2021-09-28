## tkn taskrun delete

Delete TaskRuns in a namespace

***Aliases**: rm*

### Usage

```
tkn taskrun delete
```

### Synopsis

Delete TaskRuns in a namespace

### Examples

Delete TaskRuns with names 'foo' and 'bar' in namespace 'quux':

    tkn taskrun delete foo bar -n quux

or

    tkn tr rm foo bar -n quux


### Options

```
      --all                           Delete all TaskRuns in a namespace (default: false)
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
      --clustertask string            The name of a ClusterTask whose TaskRuns should be deleted (does not delete the ClusterTask)
  -f, --force                         Whether to force deletion (default: false)
  -h, --help                          help for delete
      --keep int                      Keep n most recent number of TaskRuns
      --keep-since int                When deleting all TaskRuns keep the ones that has been completed since n minutes
  -o, --output string                 Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-as-json|jsonpath-file.
      --show-managed-fields           If true, keep the managedFields when printing objects in JSON or YAML format.
  -t, --task string                   The name of a Task whose TaskRuns should be deleted (does not delete the task)
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

* [tkn taskrun](tkn_taskrun.md)	 - Manage TaskRuns

