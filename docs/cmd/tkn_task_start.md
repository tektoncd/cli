## tkn task start

Start Tasks

### Usage

```
tkn task start [RESOURCES...] [PARAMS...] [SERVICEACCOUNT]
```

### Synopsis

Start Tasks

### Examples

Start Task foo by creating a TaskRun named "foo-run-xyz123" from namespace 'bar':

    tkn task start foo -s ServiceAccountName -n bar

The Task can either be specified by reference in a cluster using the positional argument, 
an oci bundle using the --image argument and the positional argument or in a file using the --filename argument

Authentication:
	There are three ways to authenticate against your registry when using the --image argument.
	1. By default, your docker.config in your home directory and podman's auth.json are used.
	2. Additionally, you can supply a Bearer Token via --remote-bearer
	3. Additionally, you can use Basic auth via --remote-username and --remote-password

For params values, if you want to provide multiple values, provide them comma separated
like cat,foo,bar

For passing the workspaces via flags:

- In case of emptyDir, you can pass it like -w name=my-empty-dir,emptyDir=
- In case of configMap, you can pass it like -w name=my-config,config=rpg,item=ultimav=1
- In case of secrets, you can pass it like -w name=my-secret,secret=secret-name
- In case of pvc, you can pass it like -w name=my-pvc,claimName=pvc1
- In case of volumeClaimTemplate, you can pass it like -w name=my-volume-claim-template,volumeClaimTemplateFile=workspace-template.yaml
  but before you need to create a workspace-template.yaml file. Sample contents of the file are as follows:
  spec:
   accessModes:
     - ReadWriteOnce
   resources:
     requests:
       storage: 1Gi
- In case of binding a CSI workspace, you can pass it like -w name=my-csi,csiFile=csi.yaml
  but you need to create a csi.yaml file before hand. Sample contents of the file are as follows:
  
  driver: secrets-store.csi.k8s.io
  readOnly: true
  volumeAttributes:
    secretProviderClass: "vault-database"


### Options

```
      --dry-run                   preview TaskRun without running it
  -f, --filename string           local or remote file name containing a Task definition to start a TaskRun
  -h, --help                      help for start
  -i, --image string              use an oci bundle
  -l, --labels strings            pass labels as label=value.
  -L, --last                      re-run the Task using last TaskRun values
      --output string             format of TaskRun (yaml or json)
  -p, --param stringArray         pass the param as key=value for string type, or key=value1,value2,... for array type, or key="key1:value1, key2:value2" for object type
      --pod-template string       local or remote file containing a PodTemplate definition
      --prefix-name string        specify a prefix for the TaskRun name (must be lowercase alphanumeric characters)
      --remote-bearer string      A Bearer token to authenticate against the repository
      --remote-password string    A password to pass to the registry for basic auth. Must be used with --remote-username
      --remote-skip-tls           If set to true, skips TLS check when connecting to the registry
      --remote-username string    A username to pass to the registry for basic auth. Must be used with --remote-password
  -s, --serviceaccount string     pass the serviceaccount name
      --showlog                   show logs right after starting the Task
      --skip-optional-workspace   skips the prompt for optional workspaces
      --timeout string            timeout for TaskRun
      --use-param-defaults        use default parameter values without prompting for input
      --use-taskrun string        specify a TaskRun name to use its values to re-run the TaskRun
  -w, --workspace stringArray     pass one or more workspaces to map to the corresponding physical volumes
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --no-color            disable coloring (default: false)
```

### SEE ALSO

* [tkn task](tkn_task.md)	 - Manage Tasks

