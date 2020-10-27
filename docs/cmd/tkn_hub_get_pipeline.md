## tkn hub get pipeline

Get Pipeline by name, catalog and version

### Usage

```
tkn hub get pipeline
```

### Synopsis

Get Pipeline by name, catalog and version

### Examples


Get a Pipeline of name 'foo':

    tkn hub get pipeline foo

or

Get a Pipeline of name 'foo' of version '0.3':

    tkn hub get pipeline foo --version 0.3


### Options

```
  -h, --help   help for pipeline
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL (default "https://api.hub.tekton.dev")
      --from string         Name of Catalog to which resource belongs to. (default "tekton")
      --version string      Version of Resource
```

### SEE ALSO

* [tkn hub get](tkn_hub_get.md)	 - Get resource manifest by its name, kind, catalog, and version

