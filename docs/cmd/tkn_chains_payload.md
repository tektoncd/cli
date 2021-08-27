## tkn chains payload

Print a Tekton chains' payload for a specific taskrun

### Usage

```
tkn chains payload
```

### Synopsis

Print a Tekton chains' payload for a specific taskrun

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
  -C, --no-color   disable coloring (default: false)
```

### SEE ALSO

* [tkn chains](tkn_chains.md)	 - Manage Tekton Chains

