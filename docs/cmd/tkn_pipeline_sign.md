## tkn pipeline sign

Sign Tekton Pipeline

### Usage

```
tkn pipeline sign
```

### Synopsis


	Sign the Tekton Pipeline with user provided private key file or KMS reference. Key files support ecdsa, ed25519, rsa.
	For KMS:
	* GCP, this should have the structure of gcpkms://projects/<project>/locations/<location>/keyRings/<keyring>/cryptoKeys/<key> where location, keyring, and key are filled in appropriately. Run "gcloud auth application-default login" to authenticate
	* Vault, this should have the structure of hashivault://<keyname>, where the keyname is filled out appropriately.
	* AWS, this should have the structure of awskms://[ENDPOINT]/[ID/ALIAS/ARN] (endpoint optional).
	* Azure, this should have the structure of azurekms://[VAULT_NAME][VAULT_URL]/[KEY_NAME].

### Examples

Sign a Pipeline pipeline.yaml:
	tkn pipeline sign pipeline.yaml -K=cosign.key -f=signed.yaml
or using kms
	tkn pipeline sign pipeline.yaml -m=gcpkms://projects/PROJECTID/locations/LOCATION/keyRings/KEYRING/cryptoKeys/KEY/cryptoKeyVersions/VERSION -f=signed.yaml

### Options

```
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -f, --file-name string              Fle name of the signed pipeline, using the original file name will overwrite the file
  -h, --help                          help for sign
  -K, --key-file string               Key file
  -m, --kms-key string                KMS key url
  -o, --output string                 Output format. One of: (json, yaml, name, go-template, go-template-file, template, templatefile, jsonpath, jsonpath-as-json, jsonpath-file).
      --show-managed-fields           If true, keep the managedFields when printing objects in JSON or YAML format.
      --template string               Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview].
```

### Options inherited from parent commands

```
  -c, --context string      name of the kubeconfig context to use (default: kubectl config current-context)
  -k, --kubeconfig string   kubectl config file (default: $HOME/.kube/config)
  -n, --namespace string    namespace to use (default: from $KUBECONFIG)
  -C, --no-color            disable coloring (default: false)
```

### SEE ALSO

* [tkn pipeline](tkn_pipeline.md)	 - Manage pipelines

