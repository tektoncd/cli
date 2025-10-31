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

package pipeline

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/trustedresources"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/yaml"
)

type signOptions struct {
	keyfile    string
	kmsKey     string
	targetFile string
	apiVersion string
}

func signCommand() *cobra.Command {
	opts := &signOptions{}
	f := cliopts.NewPrintFlags("trustedresources")
	eg := `Sign a Pipeline pipeline.yaml:
	tkn pipeline sign pipeline.yaml -K=cosign.key -f=signed.yaml
or using kms
	tkn pipeline sign pipeline.yaml -m=gcpkms://projects/PROJECTID/locations/LOCATION/keyRings/KEYRING/cryptoKeys/KEY/cryptoKeyVersions/VERSION -f=signed.yaml`
	long := `
	Sign the Tekton Pipeline with user provided private key file or KMS reference. Key files support ecdsa, ed25519, rsa.
	For KMS:
	* GCP, this should have the structure of gcpkms://projects/<project>/locations/<location>/keyRings/<keyring>/cryptoKeys/<key> where location, keyring, and key are filled in appropriately. Run "gcloud auth application-default login" to authenticate
	* Vault, this should have the structure of hashivault://<keyname>, where the keyname is filled out appropriately.
	* AWS, this should have the structure of awskms://[ENDPOINT]/[ID/ALIAS/ARN] (endpoint optional).
	* Azure, this should have the structure of azurekms://[VAULT_NAME][VAULT_URL]/[KEY_NAME].`
	c := &cobra.Command{
		Use:   "sign",
		Short: "Sign Tekton Pipeline",
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
				log.Fatalf("error reading file: %v", err)
				return err
			}

			var crd metav1.Object
			if opts.apiVersion == "v1beta1" {
				crd = &v1beta1.Pipeline{}
			} else {
				crd = &v1.Pipeline{}
			}

			if err := yaml.Unmarshal(b, &crd); err != nil {
				return fmt.Errorf("error unmarshalling Pipeline: %v", err)
			}

			// Sign the Pipeline and save to file
			if err := trustedresources.Sign(crd, opts.keyfile, opts.kmsKey, opts.targetFile); err != nil {
				return fmt.Errorf("error signing Pipeline: %v", err)
			}
			fmt.Fprintf(s.Out, "Pipeline %s is signed successfully \n", args[0])
			return nil
		},
	}
	f.AddFlags(c)
	c.Flags().StringVarP(&opts.keyfile, "key-file", "K", "", "Key file")
	c.Flags().StringVarP(&opts.kmsKey, "kms-key", "m", "", "KMS key url")
	c.Flags().StringVarP(&opts.targetFile, "file-name", "f", "", "Fle name of the signed pipeline, using the original file name will overwrite the file")
	c.Flags().StringVarP(&opts.apiVersion, "api-version", "", "v1", "apiVersion of the Pipeline to be signed")
	return c
}

func (s *signOptions) Run(args []string) error {
	tsBuf, err := os.ReadFile(args[0])
	if err != nil {
		log.Fatalf("error reading file: %v", err)
		return err
	}

	crd := &v1beta1.Pipeline{}
	if err := yaml.Unmarshal(tsBuf, &crd); err != nil {
		log.Fatalf("error unmarshalling Pipeline: %v", err)
		return err
	}

	// Sign the Pipeline and write to target file
	if err := trustedresources.Sign(crd, s.keyfile, s.kmsKey, s.targetFile); err != nil {
		log.Fatalf("error signing Pipeline: %v", err)
		return err
	}

	return nil
}
