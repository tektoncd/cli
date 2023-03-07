## tkn hub info

Display info of resource by its name, kind, catalog, and version

### Usage

```
tkn hub info
```

### Synopsis

Display info of resource by its name, kind, catalog, and version

### Options

```
      --from string      Name of Catalog to which resource belongs.
  -h, --help             help for info
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
* [tkn hub info task](tkn_hub_info_task.md)	 - Display info of Task by its name, catalog and version

