# pipeline start additional content

## Description

To start a pipeline, you will need to pass the following:

- Workspaces
- Parameters, at least those that have no default value

## Examples

To run a Pipeline that has one workspace and no parameter.

	$ tkn pipeline start --workspace name=my-empty-dir,emptyDir=""


To create a workspace-template.yaml

	$ cat <<EOF > workspace-template.yaml
	spec:
  		accessModes:
  			- ReadWriteOnce
  		resources:
    		requests:
      			storage: 1Gi
	EOF

To create a csi-template.yaml

	```sh
	$ cat <<EOF > csi-template.yaml
	driver: secrets-store.csi.k8s.io
	readOnly: true
	volumeAttributes:
  		secretProviderClass: "vault-database"
	EOF
	```

To run a Pipeline that has two parameters (foo and bar) and
six workspaces (my-config, my-pvc, my-secret, my-empty-dir,
my-csi-template and my-volume-claim-template)


	$ tkn pipeline start \
		--param foo=yay \
		--param bar=10 \
		--workspace name=my-secret,secret=secret-name \
		--workspace name=my-config,config=rpg,item=ultimav=1 \
		--workspace name=my-empty-dir,emptyDir="" \
		--workspace name=my-pvc,claimName=pvc1,subPath=dir
		--workspace name=my-volume-claim-template,volumeClaimTemplateFile=workspace-template.yaml
		--workspace name=my-csi-template,csiFile=csi-template.yaml