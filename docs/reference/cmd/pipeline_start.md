# pipeline start additional content

## Description

To start a pipeline, you will need to pass the following:

- Resources
- Parameters, at least those that have no default value

## Examples

To run a Pipeline that has one git resource and no parameter.

	$ tkn pipeline start --resource source=samples-git


To run a Pipeline that has one git resource, one image resource,
two parameters (foo and bar) and four workspaces (my-config, my-pvc,
my-secret and my-empty-dir)


	$ tkn pipeline start --resource source=samples-git \
		--resource image=my-image \
		--param foo=yay \
		--param bar=10 \
		--workspace name=my-secret,secret=secret-name \
		--workspace name=my-config,config=rpg,item=ultimav=1 \
		--workspace name=my-empty-dir,emptyDir="" \
		--workspace name=my-pvc,claimName=pvc1,subPath=dir
