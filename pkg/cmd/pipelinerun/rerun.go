// Copyright © 2025 The Tekton Authors.
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

package pipelinerun

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	pipelinerunpkg "github.com/tektoncd/cli/pkg/pipelinerun"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type rerunOptions struct {
	DryRun          bool
	Output          string
	PrefixName      string
	ShowLog         bool
	ExitWithPrError bool
}

func rerunCommand(p cli.Params) *cobra.Command {
	opts := &rerunOptions{}

	eg := `Rerun the PipelineRun named 'foo' from namespace 'bar':

    tkn pipelinerun rerun foo -n bar

or

    tkn pr rerun foo -n bar
`

	c := &cobra.Command{
		Use:          "rerun",
		Short:        "Rerun a PipelineRun with the same configuration",
		Example:      eg,
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		ValidArgsFunction: formatted.ParentCompletion,
		Args:              cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			prName := args[0]
			return opts.run(s, p, prName)
		},
	}

	c.Flags().BoolVarP(&opts.DryRun, "dry-run", "", false, "preview PipelineRun without running it")
	c.Flags().StringVarP(&opts.Output, "output", "o", "", "format of PipelineRun (yaml, json or name)")
	c.Flags().StringVarP(&opts.PrefixName, "prefix-name", "", "", "specify a prefix for the PipelineRun name (must be lowercase alphanumeric characters)")
	c.Flags().BoolVarP(&opts.ShowLog, "showlog", "", false, "show logs right after starting the Pipeline")
	c.Flags().BoolVarP(&opts.ExitWithPrError, "exit-with-pipelinerun-error", "E", false, "when using --showlog, exit with pipelinerun to the unix shell, 0 if success, 1 if error, 2 on unknown status")

	return c
}

func (opts *rerunOptions) run(s *cli.Stream, p cli.Params, prName string) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client: %v", err)
	}

	// Get the existing PipelineRun
	originalPr, err := pipelinerunpkg.GetPipelineRun(pipelineRunGroupResource, cs, prName, p.Namespace())
	if err != nil {
		return fmt.Errorf("failed to find PipelineRun %s: %v", prName, err)
	}

	// Create a new PipelineRun based on the original
	newPr := &v1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1",
			Kind:       "PipelineRun",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: p.Namespace(),
		},
		Spec: originalPr.Spec,
	}

	// Reset the status to empty (clear any previous cancellation or completion status)
	newPr.Spec.Status = ""

	// Set the name generation
	if opts.PrefixName != "" {
		newPr.ObjectMeta.GenerateName = opts.PrefixName + "-"
	} else {
		newPr.ObjectMeta.GenerateName = originalPr.ObjectMeta.Name + "-"
	}

	// Handle output formatting for dry-run
	if opts.DryRun {
		return opts.printDryRunResult(s, newPr)
	}

	// Create the new PipelineRun
	prv1beta1 := &v1beta1.PipelineRun{}
	err = prv1beta1.ConvertFrom(context.Background(), newPr)
	if err != nil {
		return fmt.Errorf("failed to convert PipelineRun to v1beta1: %v", err)
	}

	prCreated, err := pipelinerunpkg.Create(cs, prv1beta1, metav1.CreateOptions{}, p.Namespace())
	if err != nil {
		return fmt.Errorf("failed to create PipelineRun: %v", err)
	}

	// Convert back to v1 for display
	prCreatedV1 := &v1.PipelineRun{}
	err = prCreated.ConvertTo(context.Background(), prCreatedV1)
	if err != nil {
		return fmt.Errorf("failed to convert created PipelineRun to v1: %v", err)
	}

	// // Handle output format
	if opts.Output != "" {
		return opts.printFormattedResult(s, prCreatedV1)
	}

	// Default output
	fmt.Fprintf(s.Out, "PipelineRun started: %s\n", prCreatedV1.ObjectMeta.Name)

	if !opts.ShowLog {
		inOrderString := ""
		if prCreatedV1.Status.PipelineSpec != nil && prCreatedV1.Status.PipelineSpec.Finally != nil {
			inOrderString = "\nIn order to track the PipelineRun progress run:\ntkn pipelinerun logs %s -f -n %s\n"
		} else {
			inOrderString = "\nIn order to track the PipelineRun progress run:\ntkn pipelinerun logs %s -f -n %s\n"
		}
		fmt.Fprintf(s.Out, inOrderString, prCreatedV1.ObjectMeta.Name, p.Namespace())
	}

	// Handle showlog option
	if opts.ShowLog {
		return opts.showLogs(s, p, prCreatedV1.ObjectMeta.Name)
	}

	return nil
}

func (opts *rerunOptions) printDryRunResult(s *cli.Stream, pr *v1.PipelineRun) error {
	format := strings.ToLower(opts.Output)
	
	switch format {
	case "", "yaml":
		prBytes, err := yaml.Marshal(pr)
		if err != nil {
			return err
		}
		fmt.Fprintf(s.Out, "%s", prBytes)
	case "json":
		prBytes, err := json.Marshal(pr)
		if err != nil {
			return err
		}
		fmt.Fprintf(s.Out, "%s", prBytes)
	case "name":
		fmt.Fprintf(s.Out, "pipelinerun.tekton.dev/%s\n", pr.ObjectMeta.GenerateName)
	default:
		return fmt.Errorf("output format specified is %s but must be yaml, json or name", opts.Output)
	}
	return nil
}

func (opts *rerunOptions) printFormattedResult(s *cli.Stream, pr *v1.PipelineRun) error {
	format := strings.ToLower(opts.Output)
	
	switch format {
	case "yaml":
		prBytes, err := yaml.Marshal(pr)
		if err != nil {
			return err
		}
		fmt.Fprintf(s.Out, "%s", prBytes)
	case "json":
		prBytes, err := json.Marshal(pr)
		if err != nil {
			return err
		}
		fmt.Fprintf(s.Out, "%s", prBytes)
	case "name":
		fmt.Fprintf(s.Out, "pipelinerun.tekton.dev/%s\n", pr.ObjectMeta.Name)
	default:
		return fmt.Errorf("output format specified is %s but must be yaml, json or name", opts.Output)
	}
	return nil
}

func (opts *rerunOptions) showLogs(s *cli.Stream, p cli.Params, prName string) error {
	// This would normally delegate to the logs command
	// For now, we'll just print a message
	fmt.Fprintf(s.Out, "Use 'tkn pipelinerun logs %s -f -n %s' to follow the logs\n", prName, p.Namespace())
	return nil
}
