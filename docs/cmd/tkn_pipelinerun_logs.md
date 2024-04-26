## tkn pipelinerun logs

Show the logs of a PipelineRun

### Usage

```
tkn pipelinerun logs
```

### Synopsis

Show the logs of a PipelineRun

### Examples

Show the logs of PipelineRun named 'foo' from namespace 'bar':

    tkn pipelinerun logs foo -n bar

Show the logs of PipelineRun named 'microservice-1' for Task 'build' only from namespace 'bar':

    tkn pr logs microservice-1 -t build -n bar

Show the logs of PipelineRun named 'microservice-1' for all Tasks and steps (including init steps) from namespace 'foo':

    tkn pr logs microservice-1 -a -n foo
   

### Options

```
  -a, --all                           show all logs including init steps injected by tekton
  -E, --exit-with-pipelinerun-error   exit with pipelinerun to the unix shell, 0 if success, 1 if error, 2 on unknown status
  -f, --follow                        stream live logs
  -F, --fzf                           use fzf to select a PipelineRun
  -h, --help                          help for logs
  -L, --last                          show logs for last PipelineRun
      --limit int                     lists number of PipelineRuns (default 5)
      --prefix                        prefix each log line with the log source (task name and step name) (default true)
  -t, --task strings                  show logs for mentioned Tasks only
      --timestamps                    show logs with timestamp
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

