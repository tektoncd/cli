## tkn bundle push

Create or replace a Tekton bundle

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
	1. By default, your docker.config in your home directory and podman's auth.json are used.
	2. Additionally, you can supply a Bearer Token via --remote-bearer
	3. Additionally, you can use Basic auth via --remote-username and --remote-password

Input:
	Valid input in any form is valid Tekton YAML or JSON with a fully-specified "apiVersion" and "kind". To pass multiple objects in a single input, use "---" separators in YAML or a top-level "[]" in JSON.

Created time:
	The default created time of the OCI Image Configuration layer is set to 1970-01-01T00:00:00Z. Changing it can be done by either providing it via --ctime parameter or setting the SOURCE_DATE_EPOCH environment variable.


### Options

```
      --annotate strings         OCI Manifest annotation in the form of key=value to be added to the OCI image. Can be provided multiple times to add multiple annotations.
      --ctime string             YYYY-MM-DD, YYYY-MM-DDTHH:MM:SS or RFC3339 formatted created time to set, defaults to current time. In non RFC3339 syntax dates are in UTC timezone.
  -f, --filenames strings        List of fully-qualified file paths containing YAML or JSON defined Tekton objects to include in this bundle
  -h, --help                     help for push
      --label strings            OCI Config labels in the form of key=value to be added to the OCI image. Can be provided multiple times to add multiple labels.
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

