## tkn pipeline start

Start Pipelines

### Usage

```
tkn pipeline start
```

### Synopsis

To start a pipeline, you will need to pass the following:

- Workspaces
- Parameters, at least those that have no default value

### Examples

To run a Pipeline that has one workspace and no parameter.

	$ tkn pipeline start --workspace name=my-empty-dir,emptyDir=""


To create a workspace-template.yaml

	$ cat <<EOF > workspace-template.yaml
	spec:
  		accessModes:
  			- ReadWriteOnce
  		resources:
    		requests:
      			storage: 1Gi
	EOF

To create a csi-template.yaml

	```sh
	$ cat <<EOF > csi-template.yaml
	driver: secrets-store.csi.k8s.io
	readOnly: true
	volumeAttributes:
  		secretProviderClass: "vault-database"
	EOF
	```

To run a Pipeline that has two parameters (foo and bar) and
six workspaces (my-config, my-pvc, my-secret, my-empty-dir,
my-csi-template and my-volume-claim-template)


	$ tkn pipeline start \
		--param foo=yay \
		--param bar=10 \
		--workspace name=my-secret,secret=secret-name \
		--workspace name=my-config,config=rpg,item=ultimav=1 \
		--workspace name=my-empty-dir,emptyDir="" \
		--workspace name=my-pvc,claimName=pvc1,subPath=dir
		--workspace name=my-volume-claim-template,volumeClaimTemplateFile=workspace-template.yaml
		--workspace name=my-csi-template,csiFile=csi-template.yaml

### Options

```
      --dry-run                       preview PipelineRun without running it
  -E, --exit-with-pipelinerun-error   when using --showlog, exit with pipelinerun to the unix shell, 0 if success, 1 if error, 2 on unknown status
  -f, --filename string               local or remote file name containing a Pipeline definition to start a PipelineRun
      --finally-timeout string        timeout for Finally TaskRuns
  -h, --help                          help for start
  -l, --labels strings                pass labels as label=value.
  -L, --last                          re-run the Pipeline using last PipelineRun values
  -o, --output string                 format of PipelineRun (yaml, json or name)
  -p, --param stringArray             pass the param as key=value for string type, or key=value1,value2,... for array type, or key="key1:value1, key2:value2" for object type
      --pipeline-timeout string       timeout for PipelineRun
      --pod-template string           local or remote file containing a PodTemplate definition
      --prefix-name string            specify a prefix for the PipelineRun name (must be lowercase alphanumeric characters)
  -s, --serviceaccount string         pass the serviceaccount name
      --showlog                       show logs right after starting the Pipeline
      --skip-optional-workspace       skips the prompt for optional workspaces
      --task-serviceaccount strings   pass the service account corresponding to the task
      --tasks-timeout string          timeout for Pipeline TaskRuns
      --use-param-defaults            use default parameter values without prompting for input
      --use-pipelinerun string        use this pipelinerun values to re-run the pipeline. 
  -w, --workspace stringArray         pass one or more workspaces to map to the corresponding physical volumes
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

