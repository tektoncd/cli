## tkn pipeline start

Start pipelines

### Usage

```
tkn pipeline start pipeline [RESOURCES...] [PARAMS...] [SERVICEACCOUNT]
```

### Synopsis

To start a pipeline, you will need to pass the following:

- Resources
- Parameters, at least those that have no default value

### Examples

To run a Pipeline that has one git resource and no parameter.

	$ tkn pipeline start --resource source=samples-git


To run a Pipeline that has one git resource, one image resource and
two parameters (foo and bar)


	$ tkn pipeline start --resource source=samples-git \
		--resource image=my-image \
		--param foo=yay \
		--param bar=10

### Options

```
      --dry-run                       preview pipelinerun without running it
  -f, --filename string               local or remote file name containing a pipeline definition to start a pipelinerun
  -h, --help                          help for start
  -l, --labels strings                pass labels as label=value.
  -L, --last                          re-run the pipeline using last pipelinerun values
      --output string                 format of pipelinerun dry-run (yaml or json)
  -p, --param stringArray             pass the param as key=value or key=value1,value2
      --prefix-name string            specify a prefix for the pipelinerun name (must be lowercase alphanumeric characters)
  -r, --resource strings              pass the resource name and ref as name=ref
  -s, --serviceaccount string         pass the serviceaccount name
      --showlog                       show logs right after starting the pipeline
      --task-serviceaccount strings   pass the service account corresponding to the task
      --timeout string                timeout for pipelinerun (default "1h")
      --use-pipelinerun string        use this pipelinerun values to re-run the pipeline. 
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

