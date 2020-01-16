## tkn apply

Uses kubectl apply to create and update Kubernetes resources

### Usage

```
tkn apply
```

### Synopsis

Uses kubectl apply to create and update Kubernetes resources

### Examples

Create or update resources defined in foo.yaml in namespace 'bar':

	tkn apply -f foo.yaml -n bar


### Options

```
  -h, --help   help for apply
```

### SEE ALSO

* [tkn](tkn.md)	 - CLI for tekton pipelines

