package chains

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/taskrun"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func signatureCommand(p cli.Params) *cobra.Command {
	longHelp := ``

	c := &cobra.Command{
		Use:   "signature",
		Short: "Print a Tekton chains' signature for a specific taskrun",
		Long:  longHelp,
		Annotations: map[string]string{
			"commandType": "main",
		},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
					// The key must be fetched from the secrets.
					opts.Key = x509Keypair
				}

				// Fetch the signature.
				signatures, err := backend.RetrieveSignatures(opts)
				if err != nil {
					return fmt.Errorf("error retrieving the signature: %s", err)
				}

				// Display the signature.
				for _, signature := range signatures {
					fmt.Println(signature)
				}
			}
			return nil
		},
	}

	return c
}
