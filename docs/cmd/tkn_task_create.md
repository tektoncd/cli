## tkn task create

Create a task resource in a namespace

### Usage

```
tkn task create
```

### Synopsis

Create a task resource in a namespace

### Examples


# Create a Task defined by foo.yaml in namespace 'bar'
tkn task create -f foo.yaml -n bar


### Options

```
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -f, --from string                   Filename to use to create the resource
  -h, --help                          help for create
  -o, --output string                 Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-file.
      --template string               Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].
```

### Options inherited from parent commands

```
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
```

### SEE ALSO

* [tkn task](tkn_task.md)	 - Manage tasks

