## tkn task start

Start tasks

***Aliases**: trigger*

### Usage

```
tkn task start task [RESOURCES...] [PARAMS...] [SERVICEACCOUNT]
```

### Synopsis

Start tasks

### Examples

Start Task foo by creating a TaskRun named "foo-run-xyz123" from namespace 'bar':

    tkn task start foo -s ServiceAccountName -n bar

The rask can either be specified by reference in a cluster using the positional argument
or in a file using the --filename argument.

For params value, if you want to provide multiple values, provide them comma separated
like cat,foo,bar


### Options

```
  -f, --filename string          local or remote file name containing a task definition
  -h, --help                     help for start
  -i, --inputresource strings    pass the input resource name and ref as name=ref
  -l, --labels strings           pass labels as label=value.
  -L, --last                     re-run the task using last taskrun values
  -o, --outputresource strings   pass the output resource name and ref as name=ref
  -p, --param stringArray        pass the param as key=value or key=value1,value2
  -s, --serviceaccount string    pass the serviceaccount name
      --showlog                  show logs right after starting the task
  -t, --timeout int              timeout for taskrun in seconds (default 3600)
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --nocolour            disable colouring (default: false)
```

### SEE ALSO

* [tkn task](tkn_task.md)	 - Manage tasks

