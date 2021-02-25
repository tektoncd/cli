## tkn bundle push

Push a new Tekton bundle

***Aliases**: p*

### Usage

```
tkn bundle push
```

### Synopsis

Publish a new Tekton Bundle to a registry by passing in a set of Tekton objects via files, arguments or standard in:

	tkn bundle push docker.io/myorg/mybundle:latest "apiVersion: tekton.dev/v1beta1 kind: Pipeline..."
	tkn bundle push docker.io/myorg/mybundle:1.0 -f path/to/my/file.json
	cat path/to/my/unified_yaml_file.yaml | tkn bundle push myprivateregistry.com/myorg/mybundle -f -

Authentication:
	There are three ways to authenticate against your registry.
	1. By default, your docker.config in your home directory is used.
	2. Additionally, you can supply a Bearer Token via --remote-bearer
	3. Additionally, you can use Basic auth via --remote-username and --remote-password

Input:
	Valid input in any form is valid Tekton YAML or JSON with a fully-specified "apiVersion" and "kind". To pass multiple objects in a single input, use "---" separators in YAML or a top-level "[]" in JSON.


### Options

```
  -f, --filenames strings        List of fully-qualified file paths containing YAML or JSON defined Tekton objects to include in this bundle
  -h, --help                     help for push
      --remote-bearer string     A Bearer token to authenticate against the repository
      --remote-password string   A password to pass to the registry for basic auth. Must be used with --remote-username
      --remote-skip-tls          If set to true, skips TLS check when connecting to the registry
      --remote-username string   A username to pass to the registry for basic auth. Must be used with --remote-password
```

### Options inherited from parent commands

```
  -C, --no-color   disable coloring (default: false)
```

### SEE ALSO

* [tkn bundle](tkn_bundle.md)	 - Manage Tekton Bundles

