## tkn bundle list

List a Tekton bundle's contents

### Usage

```
tkn bundle list
```

### Synopsis

List the contents of a Tekton Bundle from a registry. You can further narrow down the results by 
optionally specifying the kind, and then the name:

	tkn bundle list docker.io/myorg/mybundle:latest // fetches all objects
	tkn bundle list docker.io/myorg/mybundle:1.0 Task // fetches all Tekton tasks
	tkn bundle list docker.io/myorg/mybundle:1.0 Task foo // fetches the Tekton task "foo"

As with other "list" commands, you can specify the desired output format using the "-o" flag.

Authentication:
	There are three ways to authenticate against your registry.
	1. By default, your docker.config in your home directory is used.
	2. Additionally, you can supply a Bearer Token via --remote-bearer
	3. Additionally, you can use Basic auth via --remote-username and --remote-password

Caching:
    By default, bundles will be cached in ~/.tekton/bundles. If you would like to use a different location, set 
"--cache-dir" and if you would like to skip the cache altogether, set "--no-cache".


### Options

```
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
      --cache-dir string              A directory to cache Tekton bundles in. (default "~/.tekton/bundles")
  -h, --help                          help for list
      --no-cache                      If set to true, pulls a Tekton bundle from the remote even its exact digest is available in the cache.
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

