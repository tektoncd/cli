## tkn taskrun describe

Describe a TaskRun in a namespace

***Aliases**: desc*

### Usage

```
tkn taskrun describe
```

### Synopsis

Describe a TaskRun in a namespace

### Examples

Describe a TaskRun of name 'foo' in namespace 'bar':

    tkn taskrun describe foo -n bar

or

    tkn tr desc foo -n bar


### Options

```
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -F, --fzf                           use fzf to select a taskrun to describe
  -h, --help                          help for describe
  -L, --last                          show description for last TaskRun
      --limit int                     lists number of TaskRuns when selecting a TaskRun to describe (default 5)
  -o, --output string                 Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-as-json|jsonpath-file.
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

* [tkn taskrun](tkn_taskrun.md)	 - Manage TaskRuns

