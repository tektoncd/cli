## tkn chain payload

Print Tekton Chains' payload for a specific taskrun

### Usage

```
tkn chain payload
```

### Synopsis

Print Tekton Chains' payload for a specific taskrun

### Options

```
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -h, --help                          help for payload
  -o, --output string                 Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-as-json|jsonpath-file.
      --show-managed-fields           If true, keep the managedFields when printing objects in JSON or YAML format.
  -S, --skip-verify                   Skip verifying the payload'signature
      --template string               Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].
```

### Options inherited from parent commands

```
      --chains-namespace string   namespace in which chains is installed (default "tekton-chains")
  -c, --context string            name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string         kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string          namespace to use (default: from $KUBECONFIG)
  -C, --no-color                  disable coloring (default: false)
```

### SEE ALSO

* [tkn chain](tkn_chain.md)	 - Manage Chains

