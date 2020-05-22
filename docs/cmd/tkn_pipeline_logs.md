## tkn pipeline logs

Show pipeline logs

### Usage

```
tkn pipeline logs
```

### Synopsis

Show pipeline logs

### Examples


Interactive mode: shows logs of the selected pipelinerun:

    tkn pipeline logs -n namespace

Interactive mode: shows logs of the selected pipelinerun of the given pipeline:

    tkn pipeline logs pipeline -n namespace

Show logs of given pipeline for last run:

    tkn pipeline logs pipeline -n namespace --last

Show logs for given pipeline and pipelinerun:

    tkn pipeline logs pipeline run -n namespace


### Options

```
  -a, --all         show all logs including init steps injected by tekton
  -f, --follow      stream live logs
  -h, --help        help for logs
  -L, --last        show logs for last run
      --limit int   lists number of pipelineruns (default 5)
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --nocolor             disable coloring (default: false)
```

### SEE ALSO

* [tkn pipeline](tkn_pipeline.md)	 - Manage pipelines

