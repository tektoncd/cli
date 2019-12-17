## tkn pipeline run

Start pipelines

***Aliases**: start,trigger*

### Usage

```
tkn pipeline run pipeline [RESOURCES...] [PARAMS...] [SERVICEACCOUNT]
```

### Synopsis

Start pipelines

### Examples


# start pipeline foo by creating a pipelinerun named "foo-run-xyz123" from the namespace "bar"
tkn pipeline run foo -s ServiceAccountName -n bar

For params value, if you want to provide multiple values, provide them comma separated
like cat,foo,bar


### Options

```
  -h, --help                          help for run
  -l, --labels strings                pass labels as label=value.
  -L, --last                          re-run the pipeline using last pipelinerun values
  -p, --param stringArray             pass the param as key=value or key=value1,value2
  -r, --resource strings              pass the resource name and ref as name=ref
  -s, --serviceaccount string         pass the serviceaccount name
      --showlog                       show logs right after starting the pipeline (default true)
      --task-serviceaccount strings   pass the service account corresponding to the task
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --nocolour            disable colouring (default: false)
```

### SEE ALSO

* [tkn pipeline](tkn_pipeline.md)	 - Manage pipelines

