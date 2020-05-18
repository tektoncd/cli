## tkn resource list

Lists pipeline resources in a namespace

***Aliases**: ls*

### Usage

```
tkn resource list
```

### Synopsis

Lists pipeline resources in a namespace

### Examples

List all PipelineResources in a namespace 'foo':

    tkn pre list -n foo


### Options

```
  -A, --all-namespaces                list pipeline resources from all namespaces
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -h, --help                          help for list
      --no-headers                    do not print column headers with output (default print column headers with output)
  -o, --output string                 Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-file.
      --template string               Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].
  -t, --type string                   Pipeline resource type
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --nocolour            disable colouring (default: false)
```

### SEE ALSO

* [tkn resource](tkn_resource.md)	 - Manage pipeline resources

