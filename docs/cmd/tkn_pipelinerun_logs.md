## tkn pipelinerun logs

Show the logs of PipelineRun

### Usage

```
tkn pipelinerun logs
```

### Synopsis

Show the logs of PipelineRun

### Examples

Show the logs of PipelineRun named 'foo' from namespace 'bar':

    tkn pipelinerun logs foo -n bar

Show the logs of PipelineRun named 'microservice-1' for task 'build' only from namespace 'bar':

    tkn pr logs microservice-1 -t build -n bar

Show the logs of PipelineRun named 'microservice-1' for all tasks and steps (including init steps) from namespace 'foo':

    tkn pr logs microservice-1 -a -n foo
   

### Options

```
  -a, --all            show all logs including init steps injected by tekton
  -f, --follow         stream live logs
  -h, --help           help for logs
      --limit int      lists number of pipelineruns (default 5)
  -t, --task strings   show logs for mentioned tasks only
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

