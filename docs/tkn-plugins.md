## Overview

The Tekton Command-Line Interface (CLI) `tkn` supports the usage of plugins, which extend its functionality beyond the built-in commands.

## Naming convention

Plugins for the Tekton Command-Line Interface (CLI) should adhere to the naming convention where their names have the prefix `tkn-`. This prefix helps distinguish plugins from regular commands within the CLI environment.

## Location

These plugins reside in specific directories on the user's system. By default, the Tekton CLI looks for plugins in the directory `~/.config/tkn/plugins`. However, users can customize the location of plugin directories by setting the `TKN_PLUGINS_DIR` environment variable. Additionally, the CLI respects the `XDG_CONFIG_HOME` environment variable, which specifies the base directory for user-specific configuration files. If set, the CLI will look for plugins in the directory ` $XDG_CONFIG_HOME/tkn/plugins`.

## Running

Running Tekton CLI plugins is straightforward. Once the plugins are installed in the designated plugin directory, users can invoke them just like any other Tekton CLI command. For example, to run a plugin named `tkn-myplugin`, users can simply type `tkn myplugin` in the terminal. The CLI will search for the plugin binary in the plugin directories and execute it if found. If the plugin is not found in the plugin directories, the CLI will fall back to executing the core Tekton CLI commands.

