## tkn clustertask create

Create a ClusterTask from Task

### Usage

```
tkn clustertask create
```

### Synopsis

Create a ClusterTask from Task

### Examples

Create a ClusterTask from Task 'foo' present in namespace 'ns':
	tkn clustertask create --from foo
or
	tkn clustertask create foobar --from=foo

### Options

```
      --from string   Create a ClusterTask from Task in a particular namespace
  -h, --help          help for create
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -C, --no-color            disable coloring (default: false)
```

### SEE ALSO

* [tkn clustertask](tkn_clustertask.md)	 - Manage ClusterTasks

