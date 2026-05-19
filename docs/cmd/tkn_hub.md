## tkn hub

Interact with artifacthub

### Usage

```
tkn hub
```

### Synopsis

Interact with artifacthub

Deprecation Notice: Tekton Hub support in CLI is being deprecated in favor of Artifact Hub.
The following commands currently only work with Tekton Hub and may support Artifact Hub in a future release:
  - check-upgrade
  - downgrade
  - get
  - info
  - reinstall
  - search
  - upgrade

Action Required: Users should migrate to Artifact Hub by using the '--type artifact' flag
with the install command. For example:
    tkn hub install task foo --type artifact --from tekton-catalog-tasks

When using '--type tekton', a deprecation warning will now be displayed.
Artifact Hub (https://artifacthub.io) will become the only supported hub in future releases.

### Options

```
      --api-server string   Hub API Server URL.
                            For artifact type: default 'https://artifacthub.io' (env: ARTIFACT_HUB_API_SERVER)
                            For tekton type (DEPRECATED): default 'https://api.hub.tekton.dev' (env: TEKTON_HUB_API_SERVER)
                            Can also be set in '$HOME/.tekton/hub-config'.
  -h, --help                help for hub
      --type string         The type of Hub from where to pull the resource. Either 'artifact' or 'tekton' (DEPRECATED: tekton type will be removed in a future release) (default "artifact")
```

### SEE ALSO

* [tkn](tkn.md)	 - CLI for tekton pipelines
* [tkn hub install](tkn_hub_install.md)	 - Install a resource from a catalog by its kind, name and version

