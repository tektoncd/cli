## tkn taskrun list

Lists TaskRuns in a namespace

***Aliases**: ls*

### Usage

```
tkn taskrun list
```

### Synopsis

Lists TaskRuns in a namespace

### Examples

List all TaskRuns in namespace 'bar':

    tkn tr list -n bar

List all TaskRuns of Task 'foo' in namespace 'bar':

    tkn taskrun list foo -n bar


### Options

```
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -h, --help                          help for list
      --label string                  A selector (label query) to filter on, supports '=', '==', and '!='
      --limit int                     limit taskruns listed (default: return all taskruns)
  -o, --output string                 Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-file.
      --reverse                       list taskruns in reverse order
      --template string               Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --nocolour            disable colouring (default: false)
```

### SEE ALSO

* [tkn taskrun](tkn_taskrun.md)	 - Manage taskruns

