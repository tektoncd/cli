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

The Task can either be specified by reference in a cluster using the positional argument
or in a file using the --filename argument.

For params values, if you want to provide multiple values, provide them comma separated
like cat,foo,bar


### Options

```
      --dry-run                  preview TaskRun without running it
  -f, --filename string          local or remote file name containing a Task definition to start a TaskRun
  -h, --help                     help for start
  -i, --inputresource strings    pass the input resource name and ref as name=ref
  -l, --labels strings           pass labels as label=value.
  -L, --last                     re-run the Task using last TaskRun values
      --output string            format of TaskRun dry-run (yaml or json)
  -o, --outputresource strings   pass the output resource name and ref as name=ref
  -p, --param stringArray        pass the param as key=value for string type, or key=value1,value2,... for array type
      --pod-template string      local or remote file containing a PodTemplate definition
      --prefix-name string       specify a prefix for the TaskRun name (must be lowercase alphanumeric characters)
  -s, --serviceaccount string    pass the serviceaccount name
      --showlog                  show logs right after starting the Task
      --timeout string           timeout for TaskRun
      --use-param-defaults       use default parameter values without prompting for input
      --use-taskrun string       specify a TaskRun name to use its values to re-run the TaskRun
  -w, --workspace stringArray    pass the workspace.
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --nocolour            disable colouring (default: false)
```

### SEE ALSO

* [tkn task](tkn_task.md)	 - Manage Tasks

