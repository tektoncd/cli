## tkn task verify

Verify Tekton Task

### Usage

```
tkn task verify
```

### Synopsis


	Verify the Tekton Task with user provided private key file or KMS reference. Key files support ecdsa, ed25519, rsa.
	For KMS:
	* GCP, this should have the structure of gcpkms://projects/<project>/locations/<location>/keyRings/<keyring>/cryptoKeys/<key> where location, keyring, and key are filled in appropriately. Run "gcloud auth application-default login" to authenticate
	* Vault, this should have the structure of hashivault://<keyname>, where the keyname is filled out appropriately.
	* AWS, this should have the structure of awskms://[ENDPOINT]/[ID/ALIAS/ARN] (endpoint optional).
	* Azure, this should have the structure of azurekms://[VAULT_NAME][VAULT_URL]/[KEY_NAME].

### Examples

Verify a Task signed.yaml:
	tkn Task verify signed.yaml -K=cosign.pub
or using kms
	tkn Task verify signed.yaml -m=gcpkms://projects/PROJECTID/locations/LOCATION/keyRings/KEYRING/cryptoKeys/KEY/cryptoKeyVersions/VERSION

### Options

```
      --allow-missing-template-keys   If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats. (default true)
  -h, --help                          help for verify
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

* [tkn task](tkn_task.md)	 - Manage Tasks

