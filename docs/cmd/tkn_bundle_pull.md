## tkn bundle pull

Push a new Tekton bundle

### Usage

```
tkn bundle pull
```

### Synopsis

Fetch a new Tekton Bundle from a registry and list the object(s) in the bundle. You can further narrow
down the results by optionally specifying the kind, and then the name:

	tkn bundle pull docker.io/myorg/mybundle:latest // fetches all objects
	tkn bundle push docker.io/myorg/mybundle:1.0 tasks // fetches all Tekton tasks
	tkn bundle push docker.io/myorg/mybundle:1.0 tasks foo // fetches the Tekton task "foo"

As with other "list" commands, you can specify the desired output format using the "-o" flag.

Authentication:
	There are three ways to authenticate against your registry.
	1. By default, your docker.config in your home directory is used.
	2. Additionally, you can supply a Bearer Token via --remote-bearer
	3. Additionally, you can use Basic auth via --remote-username and --remote-password


### Options

```
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -h, --help                          help for pull
  -o, --output string                 Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-as-json|jsonpath-file.
      --remote-bearer string          A Bearer token to authenticate against the repository
      --remote-password string        A password to pass to the registry for basic auth. Must be used with --remote-username
      --remote-skip-tls               If set to true, skips TLS check when connecting to the registry
      --remote-username string        A username to pass to the registry for basic auth. Must be used with --remote-password
      --template string               Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].
```

### Options inherited from parent commands

```
  -C, --no-color   disable coloring (default: false)
```

### SEE ALSO

* [tkn bundle](tkn_bundle.md)	 - Manage Tekton Bundles

