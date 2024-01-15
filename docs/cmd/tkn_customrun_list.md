## tkn customrun list

Lists CustomRuns in a namespace

***Aliases**: ls*

### Usage

```
tkn customrun list
```

### Synopsis

Lists CustomRuns in a namespace

### Examples

List all CustomRuns in namespace 'bar':

    tkn cr list -n bar


### Options

```
  -A, --all-namespaces                list CustomRuns from all namespaces
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -h, --help                          help for list
      --label string                  A selector (label query) to filter on, supports '=', '==', and '!='
      --limit int                     Limits the number of CustomRuns. If the limit value is 0 returns all
      --no-headers                    do not print column headers with output (default print column headers with output)
  -o, --output string                 Output format. One of: (json, yaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).
      --reverse                       list CustomRuns in reverse order
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

* [tkn customrun](tkn_customrun.md)	 - Manage CustomRuns

