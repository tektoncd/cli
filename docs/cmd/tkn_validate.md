## tkn validate

Validate Tekton resources in files without a cluster

### Usage

```
tkn validate
```

### Synopsis

Validate Tekton resources in files without a cluster

### Examples

Validate the Tekton resources in a file:

    tkn validate -f pipeline.yaml

Validate several files at once:

    tkn validate -f task.yaml -f pipeline.yaml

Validate resources read from standard input:

    cat pipeline.yaml | tkn validate -f -


### Options

```
  -f, --filenames strings   Files containing Tekton resources in YAML or JSON to validate, use - for standard input
  -h, --help                help for validate
```

### SEE ALSO

* [tkn](tkn.md)	 - CLI for tekton pipelines

