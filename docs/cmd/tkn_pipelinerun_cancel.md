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
      --grace string   Gracefully cancel a PipelineRun
                       To use this, you need to change the feature-flags configmap enable-api-fields to alpha instead of stable.
                       Set to 'CancelledRunFinally' if you want to cancel the current running task and directly run the finally tasks.
                       Set to 'StoppedRunFinally' if you want to cancel the remaining non-final task and directly run the finally tasks.
                       
  -h, --help           help for cancel
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

