## tkn task delete

Delete Tasks in a namespace

***Aliases**: rm*

### Usage

```
tkn task delete
```

### Synopsis

Delete Tasks in a namespace

### Examples

Delete Tasks with names 'foo' and 'bar' in namespace 'quux':

    tkn task delete foo bar -n quux

or

    tkn t rm foo bar -n quux

Delete a Task and print the result as JSON:

    tkn task delete foo -n quux -o json

Delete a Task and print the result as YAML:

    tkn task delete foo -n quux -o yaml

Delete all Tasks in a namespace and print the result as JSON:

    tkn task delete --all -n quux -o json

Delete all Tasks in a namespace and print the result as YAML:

    tkn task delete --all -n quux -o yaml

Delete a Task and its TaskRuns and print the result as JSON:

    tkn task delete foo -n quux --trs -o json

Delete a Task and its TaskRuns and print the result as YAML:

    tkn task delete foo -n quux --trs -o yaml

Using -o json or -o yaml skips the confirmation prompt.


### Options

```
      --all             Delete all Tasks in a namespace (default: false)
  -f, --force           Whether to force deletion (default: false)
  -h, --help            help for delete
  -o, --output string   Output format. One of: json|yaml. Skips the confirmation prompt
      --trs             Whether to delete Task(s) and related resources (TaskRuns) (default: false)
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

