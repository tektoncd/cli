## tkn pipelinerun cancel

Cancel a PipelineRun in a namespace

### Usage

```
tkn pipelinerun cancel
```

### Synopsis

Cancel a PipelineRun in a namespace

### Examples

Cancel the PipelineRun named 'foo' from namespace 'bar':

    tkn pipelinerun cancel foo -n bar


### Options

```
  -h, --help   help for cancel
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --nocolour            disable colouring (default: false)
```

### SEE ALSO

* [tkn pipelinerun](tkn_pipelinerun.md)	 - Manage pipelineruns

