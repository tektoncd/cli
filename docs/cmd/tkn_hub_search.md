## tkn hub search

Search resource by a combination of name, kind, and tags

### Usage

```
tkn hub search
```

### Synopsis

Search resource by a combination of name, kind, and tags

### Examples


Search a resource of name 'foo':

    tkn hub search foo

or

Search resources using tag 'cli':

    tkn hub search --tags cli


### Options

```
  -h, --help                help for search
      --kinds stringArray   Accepts a comma separated list of kinds
  -l, --limit uint          Max number of resources to fetch
      --match string        Accept type of search. 'exact' or 'contains'. (default "contains")
  -o, --output string       Accepts output format: [table, json] (default "table")
      --tags stringArray    Accepts a comma separated list of tags
```

### Options inherited from parent commands

```
      --api-server string   Hub API Server URL (default "https://api.hub.tekton.dev")
```

### SEE ALSO

* [tkn hub](tkn_hub.md)	 - Interact with tekton hub

