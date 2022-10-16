## tkn pipeline logs

Show Pipeline logs

### Usage

```
tkn pipeline logs
```

### Synopsis

Show Pipeline logs

### Examples


Interactive mode: shows logs of the selected PipelineRun:

    tkn pipeline logs -n namespace

Interactive mode: shows logs of the selected PipelineRun of the given Pipeline:

    tkn pipeline logs pipeline -n namespace

Show logs of given Pipeline for last run:

    tkn pipeline logs pipeline -n namespace --last

Show logs for given Pipeline and PipelineRun:

    tkn pipeline logs pipeline run -n namespace


### Options

```
  -a, --all          show all logs including init steps injected by tekton
  -f, --follow       stream live logs
  -h, --help         help for logs
  -L, --last         show logs for last PipelineRun
      --limit int    lists number of PipelineRuns (default 5)
  -t, --timestamps   show logs with timestamp
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --no-color            disable coloring (default: false)
```

### SEE ALSO

* [tkn pipeline](tkn_pipeline.md)	 - Manage pipelines

