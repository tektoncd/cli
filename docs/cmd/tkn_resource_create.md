## tkn resource create

Create a pipeline resource in a namespace

### Usage

```
tkn resource create
```

### Synopsis

Create a pipeline resource in a namespace

### Examples


  # Creates new PipelineResource as per the given input
	tkn resource create -n namespace
	
  # Create a PipelineResource defined by foo.yaml in namespace 'bar'
	tkn resource create -f foo.yaml -n bar

### Options

```
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -f, --from string                   local or remote filename to use to create the pipeline resource
  -h, --help                          help for create
  -o, --output string                 Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-file.
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

* [tkn resource](tkn_resource.md)	 - Manage pipeline resources

