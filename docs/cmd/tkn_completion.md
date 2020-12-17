## tkn completion

Prints shell completion scripts

### Usage

```
tkn completion [SHELL]
```

### Synopsis

This command prints shell completion code which must be evaluated to provide
interactive completion

Supported Shells:
	- bash
	- zsh


### Examples

To load completions:

Bash:

$ source <(tkn completion bash)

# To load completions for each session, execute once:
Linux:
  $ tkn completion bash > /etc/bash_completion.d/tkn

MacOS:
  $ tkn completion bash > /usr/local/etc/bash_completion.d/tkn

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ tkn completion zsh > "${fpath[1]}/_tkn"

# You will need to start a new shell for this setup to take effect.

Fish:

$ tkn completion fish | source

# To load completions for each session, execute once:
$ tkn completion fish > ~/.config/fish/completions/tkn.fish


### Options

```
  -h, --help   help for completion
```

### SEE ALSO

* [tkn](tkn.md)	 - CLI for tekton pipelines

