## tkn hub search

Search resource by a combination of name, kind, categories, platforms, and tags

### Usage

```
tkn hub search
```

### Synopsis

Search resource by a combination of name, kind, categories, platforms, and tags

### Examples


Search a resource of name 'foo':

    tkn hub search foo

or

Search resources using tag 'cli':

    tkn hub search --tags cli


### Options

```
      --categories stringArray   Accepts a comma separated list of categories
  -h, --help                     help for search
      --kinds stringArray        Accepts a comma separated list of kinds
  -l, --limit uint               Max number of resources to fetch
      --match string             Accept type of search. 'exact' or 'contains'. (default "contains")
  -o, --output string            Accepts output format: [table, json, wide] (default "table")
      --platforms stringArray    Accepts a comma separated list of platforms
      --tags stringArray         Accepts a comma separated list of tags
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL (default 'https://api.hub.tekton.dev' for 'tekton' type; default 'https://artifacthub.io' for 'artifact' type).
                            URL can also be defined in a file '$HOME/.tekton/hub-config' with a variable 'TEKTON_HUB_API_SERVER'/'ARTIFACT_HUB_API_SERVER'.
      --type string         The type of Hub from where to pull the resource. Either 'artifact' or 'tekton' (default "tekton")
```

### SEE ALSO

* [tkn hub](tkn_hub.md)	 - Interact with tekton hub

