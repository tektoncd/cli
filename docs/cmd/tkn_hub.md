## tkn hub

Interact with tekton hub

### Usage

```
tkn hub
```

### Synopsis

Interact with tekton hub

### Options

```
      --api-server string   Hub API Server URL (default 'https://api.hub.tekton.dev' for 'tekton' type; default 'https://artifacthub.io' for 'artifact' type).
                            URL can also be defined in a file '$HOME/.tekton/hub-config' with a variable 'TEKTON_HUB_API_SERVER'/'ARTIFACT_HUB_API_SERVER'.
  -h, --help                help for hub
      --type string         The type of Hub from where to pull the resource. Either 'artifact' or 'tekton' (default "tekton")
```

### SEE ALSO

* [tkn](tkn.md)	 - CLI for tekton pipelines
* [tkn hub check-upgrade](tkn_hub_check-upgrade.md)	 - Check for upgrades of resources if present
* [tkn hub downgrade](tkn_hub_downgrade.md)	 - Downgrade an installed resource
* [tkn hub get](tkn_hub_get.md)	 - Get resource manifest by its name, kind, catalog, and version
* [tkn hub info](tkn_hub_info.md)	 - Display info of resource by its name, kind, catalog, and version
* [tkn hub install](tkn_hub_install.md)	 - Install a resource from a catalog by its kind, name and version
* [tkn hub reinstall](tkn_hub_reinstall.md)	 - Reinstall a resource by its kind and name
* [tkn hub search](tkn_hub_search.md)	 - Search resource by a combination of name, kind, categories, platforms, and tags
* [tkn hub upgrade](tkn_hub_upgrade.md)	 - Upgrade an installed resource

