## tkn triggertemplate list

Lists triggertemplates in a namespace

***Aliases**: ls*

### Usage

```
tkn triggertemplate list
```

### Synopsis

Lists triggertemplates in a namespace

### Examples

List all triggertemplates in namespace 'bar':

	tkn triggertemplate list -n bar

or

	tkn tt ls -n bar


### Options

```
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -h, --help                          help for list
  -o, --output string                 Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-file.
      --template string               Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --nocolor             disable coloring (default: false)
```

### SEE ALSO

* [tkn triggertemplate](tkn_triggertemplate.md)	 - Manage triggertemplates

