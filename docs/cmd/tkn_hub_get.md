## tkn hub get

Get resource manifest by its name, kind, catalog, and version

### Usage

```
tkn hub get
```

### Synopsis

Get resource manifest by its name, kind, catalog, and version

### Options

```
      --from string      Name of Catalog to which resource belongs to.
  -h, --help             help for get
      --version string   Version of Resource
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL (default 'https://api.hub.tekton.dev' for 'tekton' type; default 'https://artifacthub.io' for 'artifact' type).
                            URL can also be defined in a file '$HOME/.tekton/hub-config' with a variable 'TEKTON_HUB_API_SERVER'/'ARTIFACT_HUB_API_SERVER'.
      --type string         The type of Hub from where to pull the resource. Either 'artifact' or 'tekton' (default "tekton")
```

### SEE ALSO

* [tkn hub](tkn_hub.md)	 - Interact with tekton hub
* [tkn hub get pipeline](tkn_hub_get_pipeline.md)	 - Get Pipeline by name, catalog and version
* [tkn hub get task](tkn_hub_get_task.md)	 - Get Task by name, catalog and version

