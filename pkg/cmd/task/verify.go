// Copyright © 2022 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package task

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/trustedresources"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/yaml"
)

type verifyOptions struct {
	keyfile string
	kmsKey  string
}

func verifyCommand() *cobra.Command {
	opts := &verifyOptions{}
	f := cliopts.NewPrintFlags("trustedresources")
	eg := `Verify a Task signed.yaml:
	tkn Task verify signed.yaml -K=cosign.pub
or using kms
	tkn Task verify signed.yaml -m=gcpkms://projects/PROJECTID/locations/LOCATION/keyRings/KEYRING/cryptoKeys/KEY/cryptoKeyVersions/VERSION`
	long := `
	Verify the Tekton Task with user provided private key file or KMS reference. Key files support ecdsa, ed25519, rsa.
	For KMS:
	* GCP, this should have the structure of gcpkms://projects/<project>/locations/<location>/keyRings/<keyring>/cryptoKeys/<key> where location, keyring, and key are filled in appropriately. Run "gcloud auth application-default login" to authenticate
	* Vault, this should have the structure of hashivault://<keyname>, where the keyname is filled out appropriately.
	* AWS, this should have the structure of awskms://[ENDPOINT]/[ID/ALIAS/ARN] (endpoint optional).
	* Azure, this should have the structure of azurekms://[VAULT_NAME][VAULT_URL]/[KEY_NAME].`
	c := &cobra.Command{
		Use:   "verify",
		Short: "Verify Tekton Task",
		Long:  long,
		Annotations: map[string]string{
			"commandType":  "main",
			"kubernetes":   "false",
			"experimental": "",
		},
		Args:    cobra.ExactArgs(1),
		Example: eg,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}
			b, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("error reading file: %v", err)
			}

			crd := &v1beta1.Task{}
			if err := yaml.Unmarshal(b, &crd); err != nil {
				return fmt.Errorf("error unmarshalling Task: %v", err)
			}

			if err := trustedresources.Verify(crd, opts.keyfile, opts.kmsKey); err != nil {
				return fmt.Errorf("error verifying Task: %v", err)
			}
			fmt.Fprintf(s.Out, "Task %s passes verification \n", args[0])
			return nil
		},
	}
	f.AddFlags(c)
	c.Flags().StringVarP(&opts.keyfile, "key-file", "K", "", "Key file")
	c.Flags().StringVarP(&opts.kmsKey, "kms-key", "m", "", "KMS key url")
	return c
}
