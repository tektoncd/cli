## tkn pipelinerun watch

Watch a PipelineRun until it finishes

### Usage

```
tkn pipelinerun watch
```

### Synopsis

Watch a PipelineRun until it finishes

### Examples

watch a PipelineRun of name 'foo' in namespace 'bar':

    tkn pipelinerun watch foo -n bar

this command will wait for the pipelinerun to finish and exit with the exit status of the
pipelinerun. 
    


### Options

```
  -h, --help      help for watch
  -L, --last      watch last PipelineRun
      --showlog   show logs of the PipelineRun
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

