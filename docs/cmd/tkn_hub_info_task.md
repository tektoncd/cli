## tkn hub info task

Display info of Task by its name, catalog and version

### Usage

```
tkn hub info task
```

### Synopsis

Display info of Task by its name, catalog and version

### Examples


Display info of a Task of name 'foo':

    tkn hub info task foo

or

Display info of a Task of name 'foo' of version '0.3':

    tkn hub info task foo --version 0.3


### Options

```
  -h, --help   help for task
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL (default 'https://api.hub.tekton.dev' for 'tekton' type; default 'https://artifacthub.io' for 'artifact' type).
                            URL can also be defined in a file '$HOME/.tekton/hub-config' with a variable 'TEKTON_HUB_API_SERVER'/'ARTIFACT_HUB_API_SERVER'.
      --from string         Name of Catalog to which resource belongs.
      --type string         The type of Hub from where to pull the resource. Either 'artifact' or 'tekton' (default "tekton")
      --version string      Version of Resource
```

### SEE ALSO

* [tkn hub info](tkn_hub_info.md)	 - Display info of resource by its name, kind, catalog, and version

