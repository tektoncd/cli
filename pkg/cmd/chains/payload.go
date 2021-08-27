package chains

import (
	"fmt"

	"github.com/spf13/cobra"
	chainslib "github.com/tektoncd/chains/pkg/chains"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/taskrun"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

type PayloadOptions struct {
	SkipVerify bool
}

func payloadCommand(p cli.Params) *cobra.Command {
	opts := &PayloadOptions{}
	f := cliopts.NewPrintFlags("chains")
	longHelp := ``

	c := &cobra.Command{
		Use:   "payload",
		Short: "Print a Tekton chains' payload for a specific taskrun",
		Long:  longHelp,
		Annotations: map[string]string{
			"commandType": "main",
		},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			skipVerify, _ := cmd.LocalFlags().GetBool("skip-verify")

			// Get the task name.
			taskName := args[0]

			// Get the Tekton clients.
			cs, err := p.Clients()
			if err != nil {
				return fmt.Errorf("failed to create tekton client")
			}

			// Retrieve the taskrun.
			tr, err := taskrun.Get(cs, taskName, metav1.GetOptions{}, p.Namespace())
			if err != nil {
				return fmt.Errorf("failed to get TaskRun %s: %v", taskName, err)
			}

			// Get the storage backend.
			backends, opts, err := GetTaskRunBackends(p, tr)
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
					opts.Key = x509Keypair
				}

				// Fetch the payload.
				payloads, err := backend.RetrievePayloads(opts)
				if err != nil {
					return fmt.Errorf("error retrieving the payloads: %s", err)
				}

				// Verify the payload signature.
				if !skipVerify && backend.Type() != "oci" {
					// Retrieve a context with the configuration.
					ctx, err := ConfigMapToContext(cs)
					if err != nil {
						return err
					}
					trv := chainslib.TaskRunVerifier{
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
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.SkipVerify, "skip-verify", "S", opts.SkipVerify, "Skip verifying the payload'signature")

	return c
}
