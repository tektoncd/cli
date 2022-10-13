## tkn pipeline

Manage pipelines

***Aliases**: p,pipelines*

### Usage

```
tkn pipeline
```

### Synopsis

Manage pipelines

### Options

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -h, --help                help for pipeline
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --no-color            disable coloring (default: false)
```

### SEE ALSO

* [tkn](tkn.md)	 - CLI for tekton pipelines
* [tkn pipeline delete](tkn_pipeline_delete.md)	 - Delete Pipelines in a namespace
* [tkn pipeline describe](tkn_pipeline_describe.md)	 - Describes a Pipeline in a namespace
* [tkn pipeline export](tkn_pipeline_export.md)	 - Export Pipeline
* [tkn pipeline list](tkn_pipeline_list.md)	 - Lists Pipelines in a namespace
* [tkn pipeline logs](tkn_pipeline_logs.md)	 - Show Pipeline logs
* [tkn pipeline sign](tkn_pipeline_sign.md)	 - Sign Tekton Pipeline
* [tkn pipeline start](tkn_pipeline_start.md)	 - Start Pipelines
* [tkn pipeline verify](tkn_pipeline_verify.md)	 - Verify Tekton Pipeline

