## tkn hub install task

Install Task from a catalog by its name and version

### Usage

```
tkn hub install task
```

### Synopsis

Install Task from a catalog by its name and version

### Examples


Install a Task of name 'foo':

    tkn hub install task foo

or

Install a Task of name 'foo' of version '0.3' from Catalog 'Tekton':

    tkn hub install task foo --version 0.3 --from tekton

Note that the resources in Artifact Hub follow full SemVer - <major>.<minor>.<patch> (e.g. 0.3.0),
please double check the version used


### Options

```
  -h, --help   help for task
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL (default 'https://api.hub.tekton.dev' for 'tekton' type; default 'https://artifacthub.io' for 'artifact' type).
                            URL can also be defined in a file '$HOME/.tekton/hub-config' with a variable 'TEKTON_HUB_API_SERVER'/'ARTIFACT_HUB_API_SERVER'.
  -c, --context string      Name of the kubeconfig context to use (default: kubectl config current-context)
      --from string         Name of Catalog to which resource belongs.
  -k, --kubeconfig string   Kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    Namespace to use (default: from $KUBECONFIG)
      --type string         The type of Hub from where to pull the resource. Either 'artifact' or 'tekton' (default "tekton")
      --version string      Version of Resource
```

### SEE ALSO

* [tkn hub install](tkn_hub_install.md)	 - Install a resource from a catalog by its kind, name and version

