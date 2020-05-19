## tkn clustertask start

Start clustertasks

### Usage

```
tkn clustertask start clustertask [RESOURCES...] [PARAMS...] [SERVICEACCOUNT]
```

### Synopsis

Start clustertasks

### Examples

Start ClusterTask foo by creating a TaskRun named "foo-run-xyz123" in namespace 'bar':

    tkn clustertask start foo -n bar

or

    tkn ct start foo -n bar

For params value, if you want to provide multiple values, provide them comma separated
like cat,foo,bar


### Options

```
      --dry-run                  preview taskrun without running it
  -h, --help                     help for start
  -i, --inputresource strings    pass the input resource name and ref as name=ref
  -l, --labels strings           pass labels as label=value.
  -L, --last                     re-run the clustertask using last taskrun values
      --output string            format of taskrun dry-run (yaml or json)
  -o, --outputresource strings   pass the output resource name and ref as name=ref
  -p, --param stringArray        pass the param as key=value for string type, or key=value1,value2,... for array type
  -s, --serviceaccount string    pass the serviceaccount name
      --showlog                  show logs right after starting the clustertask
      --timeout string           timeout for taskrun
  -w, --workspace stringArray    pass the workspace.
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -C, --nocolour            disable colouring (default: false)
```

### SEE ALSO

* [tkn clustertask](tkn_clustertask.md)	 - Manage clustertasks

