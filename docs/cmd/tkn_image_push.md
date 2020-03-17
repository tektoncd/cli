## tkn image push

Build and push an OCI image

### Usage

```
tkn image push
```

### Synopsis

Build and push an OCI image

### Examples

Push an OCI image composed of Tekton resources. Each file must contain a single Tekton resource:

    tkn image push [IMAGE REFERENCE] -f task.yaml -f pipeline.yaml


### Options

```
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -f, --file strings                  local or remote filename to add to the image
  -h, --help                          help for push
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

* [tkn image](tkn_image.md)	 - Upload OCI images

