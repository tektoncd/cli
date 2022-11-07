## tkn pipelinerun export

Export PipelineRun

### Usage

```
tkn pipelinerun export
```

### Synopsis

Export PipelineRun

### Examples

Export PipelineRun Definition:

	tkn pipelinerun export will export a pipelinerun definition as yaml to be easily
	imported or modified.

	Example: export a PipelineRun named 'pipelinerun' in namespace 'foo' and recreate
	it in the namespace 'bar':

    tkn pr export pipelinerun -n foo|kubectl create -f- -n bar


### Options

```
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -h, --help                          help for export
  -o, --output string                 Output format. One of: (json, yaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).
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

* [tkn pipelinerun](tkn_pipelinerun.md)	 - Manage PipelineRuns

