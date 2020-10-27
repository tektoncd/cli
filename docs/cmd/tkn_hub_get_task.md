## tkn hub get task

Get Task by name, catalog and version

### Usage

```
tkn hub get task
```

### Synopsis

Get Task by name, catalog and version

### Examples


Get a Task of name 'foo':

    tkn hub get task foo

or

Get a Task of name 'foo' of version '0.3':

    tkn hub get task foo --version 0.3


### Options

```
  -h, --help   help for task
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL (default "https://api.hub.tekton.dev")
      --from string         Name of Catalog to which resource belongs to. (default "tekton")
      --version string      Version of Resource
```

### SEE ALSO

* [tkn hub get](tkn_hub_get.md)	 - Get resource manifest by its name, kind, catalog, and version

