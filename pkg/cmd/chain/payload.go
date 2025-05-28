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

package chain

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/chain"
	"github.com/tektoncd/cli/pkg/cli"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PayloadOptions struct {
	SkipVerify bool
}

func payloadCommand(p cli.Params) *cobra.Command {
	opts := &PayloadOptions{}

	c := &cobra.Command{
		Use:   "payload",
		Short: "Print Tekton Chains' payload for a specific taskrun",
		Annotations: map[string]string{
			"commandType": "main",
		},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skipVerify, _ := cmd.LocalFlags().GetBool("skip-verify")

			// Get the task name.
			taskName := args[0]

			chainsNamespace, err := cmd.Flags().GetString("chains-namespace")
			if err != nil {
				return fmt.Errorf("error: output option not set properly: %v", err)
			}

			// Get the Tekton clients.
			cs, err := p.Clients()
			if err != nil {
				return fmt.Errorf("failed to create tekton client")
			}

			// Retrieve the taskrun.
			var taskrun *v1.TaskRun
			if err = actions.GetV1(taskrunGroupResource, cs, taskName, p.Namespace(), metav1.GetOptions{}, &taskrun); err != nil {
				return fmt.Errorf("failed to get TaskRun %s: %v", taskName, err)
			}

			return printPayloads(cs, chainsNamespace, taskrun, skipVerify)
		},
	}
	c.Flags().BoolVarP(&opts.SkipVerify, "skip-verify", "S", opts.SkipVerify, "Skip verifying the payload'signature")
	c.Deprecated = "The Chain command is deprecated and will be removed in future releases."

	return c
}

func printPayloads(cs *cli.Clients, namespace string, tr *v1.TaskRun, skipVerify bool) error {
	// Get the storage backend.
	backends, opts, err := chain.GetTaskRunBackends(cs, namespace, tr)
	if err != nil {
		return fmt.Errorf("failed to retrieve the backend storage: %v", err)
	}
	for _, backend := range backends {
		// Some limitations occur when the backend is OCI.
		if backend.Type() == "oci" {
			// Cannot skip the verification since it is required when fetching the
			// artifacts from the OCI registry.
			if skipVerify {
				return fmt.Errorf("verification is mandatory when the backend is OCI - remove the the `-S` flag from the command line and try again")
			}

			// The key must be fetched from the secrets.
			opts.FullKey = fmt.Sprintf(x509Keypair, namespace)
		}

		// Fetch the payload.
		trObj := objects.NewTaskRunObjectV1(tr)
		payloads, err := backend.RetrievePayloads(context.Background(), trObj, opts)
		if err != nil {
			return fmt.Errorf("error retrieving the payloads: %s", err)
		}
		if len(payloads) == 0 {
			fmt.Printf("No payloads found for taskrun %s\n", tr.Name)
			return nil
		}

		// Verify the payload signature.
		if !skipVerify && backend.Type() != "oci" {
			// Retrieve a context with the configuration.
			ctx, err := chain.ConfigMapToContext(cs, namespace)
			if err != nil {
				return err
			}
			trv := chains.TaskRunVerifier{
				KubeClient:        cs.Kube,
				Pipelineclientset: cs.Tekton,
				SecretPath:        "",
			}
			if err := trv.VerifyTaskRun(ctx, tr); err != nil {
				return err
			}
		}

		// Display the payload.
		for _, payload := range payloads {
			fmt.Println(payload)
		}
	}
	return nil
}
