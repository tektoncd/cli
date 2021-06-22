## tkn task create

Create a Task from ClusterTask

### Usage

```
tkn task create
```

### Synopsis

Create a Task from ClusterTask

### Examples

Create a Task from ClusterTask 'foo' in namespace 'ns':
	tkn task create --from foo
or
	tkn task create foobar --from=foo -n ns

### Options

```
      --from string   Create a ClusterTask from Task in a particular namespace
  -h, --help          help for create
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

