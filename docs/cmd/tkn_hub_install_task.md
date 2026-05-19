## tkn hub install task

Install Task from a catalog by its name and version

### Usage

```
tkn hub install task
```

### Synopsis

Install Task from a catalog by its name and version

### Examples


Install a Task of name 'foo' from Artifact Hub:

    tkn hub install task foo --type artifact --from tekton-catalog-tasks --version 0.3.0

or

Install a Task of name 'foo' from Tekton Hub (DEPRECATED):

    tkn hub install task foo --version 0.3 --from tekton

Note: Tekton Hub is deprecated. Use '--type artifact' with Artifact Hub instead.
Resources in Artifact Hub follow full SemVer - <major>.<minor>.<patch> (e.g. 0.3.0).


### Options

```
  -h, --help   help for task
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL.
                            For artifact type: default 'https://artifacthub.io' (env: ARTIFACT_HUB_API_SERVER)
                            For tekton type (DEPRECATED): default 'https://api.hub.tekton.dev' (env: TEKTON_HUB_API_SERVER)
                            Can also be set in '$HOME/.tekton/hub-config'.
  -c, --context string      Name of the kubeconfig context to use (default: kubectl config current-context)
      --from string         Name of Catalog to which resource belongs.
  -k, --kubeconfig string   Kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    Namespace to use (default: from $KUBECONFIG)
      --type string         The type of Hub from where to pull the resource. Either 'artifact' or 'tekton' (DEPRECATED: tekton type will be removed in a future release) (default "artifact")
      --version string      Version of Resource
```

### SEE ALSO

* [tkn hub install](tkn_hub_install.md)	 - Install a resource from a catalog by its kind, name and version

