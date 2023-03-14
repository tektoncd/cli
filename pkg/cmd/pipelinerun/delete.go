// Copyright © 2019 The Tekton Authors.
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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/deleter"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	prsort "github.com/tektoncd/cli/pkg/pipelinerun/sort"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

// prExists validates that the arguments are valid PipelineRun names
func prExists(args []string, p cli.Params) ([]string, error) {

	availablePrs := make([]string, 0)
	c, err := p.Clients()
	if err != nil {
		return availablePrs, err
	}
	var errorList error
	ns := p.Namespace()
	for _, name := range args {
		var pr *v1.PipelineRun
		err := actions.GetV1(pipelineRunGroupResource, c, name, ns, metav1.GetOptions{}, &pr)
		if err != nil {
			errorList = multierr.Append(errorList, err)
			continue
		}
		availablePrs = append(availablePrs, name)
	}
	return availablePrs, errorList
}

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "PipelineRun", ForceDelete: false, ParentResource: "Pipeline", DeleteAllNs: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete PipelineRuns with names 'foo' and 'bar' in namespace 'quux':

    tkn pipelinerun delete foo bar -n quux

or

    tkn pr rm foo bar -n quux
`

	c := &cobra.Command{
		Use:               "delete",
		Aliases:           []string{"rm"},
		Short:             "Delete PipelineRuns in a namespace",
		Example:           eg,
		ValidArgsFunction: formatted.ParentCompletion,
		Args:              cobra.MinimumNArgs(0),
		SilenceUsage:      true,

		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				In:  cmd.InOrStdin(),
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if opts.Keep < 0 && opts.KeepSince < 0 {
				return fmt.Errorf("keep and keep-since option should not be lower than 0")
			}

			if opts.Keep < 0 {
				return fmt.Errorf("keep option should not be lower than 0")
			}

			if opts.KeepSince < 0 {
				return fmt.Errorf("keep-since option should not be lower than 0")
			}

			if (opts.Keep > 0 || opts.KeepSince > 0) && opts.ParentResourceName == "" {
				opts.DeleteAllNs = true
			}

			if (opts.Keep > 0 || opts.KeepSince > 0) && opts.DeleteAllNs && opts.ParentResourceName != "" {
				return fmt.Errorf("--keep or --keep-since, --all and --%s cannot be used together", strings.ToLower(opts.ParentResource))
			}

			availablePrs, errs := prExists(args, p)
			if len(availablePrs) == 0 && errs != nil {
				return errs
			}

			if err := opts.CheckOptions(s, availablePrs, p.Namespace()); err != nil {
				return err
			}

			if err := deletePipelineRuns(s, p, availablePrs, opts); err != nil {
				return err
			}
			return errs
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().StringVarP(&opts.ParentResourceName, "pipeline", "p", "", "The name of a Pipeline whose PipelineRuns should be deleted (does not delete the Pipeline)")
	c.Flags().IntVarP(&opts.Keep, "keep", "", 0, "Keep n most recent number of PipelineRuns")
	c.Flags().IntVarP(&opts.KeepSince, "keep-since", "", 0, "When deleting all PipelineRuns keep the ones that has been completed since n minutes")
	c.Flags().BoolVarP(&opts.IgnoreRunning, "ignore-running", "i", true, "ignore running PipelineRun (default: true)")
	c.Flags().BoolVarP(&opts.DeleteAllNs, "all", "", false, "Delete all PipelineRuns in a namespace (default: false)")
	c.Flags().StringVarP(&opts.LabelSelector, "label", "", opts.LabelSelector, "A selector (label query) to filter on when running with --all, supports '=', '==', and '!='")
	return c
}

func deletePipelineRuns(s *cli.Stream, p cli.Params, prNames []string, opts *options.DeleteOptions) error {
	var numberOfDeletedPr, numberOfKeptPr int
	prGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "pipelineruns"}

	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}
	var d *deleter.Deleter
	switch {
	case opts.DeleteAllNs:
		d = deleter.New("PipelineRun", func(pipelineRunName string) error {
			return actions.Delete(prGroupResource, cs.Dynamic, cs.Tekton.Discovery(), pipelineRunName, p.Namespace(), metav1.DeleteOptions{})
		})
		prtodelete, prtokeep, err := allPipelineRunNames(cs, opts.Keep, opts.KeepSince, opts.IgnoreRunning, opts.LabelSelector, p.Namespace())
		if err != nil {
			return err
		}
		numberOfDeletedPr = len(prtodelete)
		numberOfKeptPr = len(prtokeep)
		d.Delete(s, prtodelete)
	case opts.ParentResourceName == "":
		d = deleter.New("PipelineRun", func(pipelineRunName string) error {
			return actions.Delete(prGroupResource, cs.Dynamic, cs.Tekton.Discovery(), pipelineRunName, p.Namespace(), metav1.DeleteOptions{})
		})
		d.Delete(s, prNames)
	default:
		d = deleter.New("Pipeline", func(_ string) error {
			return errors.New("the Pipeline should not be deleted")
		})

		// Create a LabelSelector to filter the PipelineRuns which are associated
		// with a particular Pipeline
		labelSelector := fmt.Sprintf("tekton.dev/pipeline=%s", opts.ParentResourceName)

		// Compute the total no of PipelineRuns which we need to delete
		prtodelete, prtokeep, err := allPipelineRunNames(cs, opts.Keep, opts.KeepSince, opts.IgnoreRunning, labelSelector, p.Namespace())
		if err != nil {
			return err
		}
		numberOfDeletedPr = len(prtodelete)
		numberOfKeptPr = len(prtokeep)

		// Delete the PipelineRuns associated with a Pipeline
		d.WithRelated("PipelineRun", pipelineRunLister(cs, opts.Keep, opts.KeepSince, p.Namespace(), opts.IgnoreRunning), func(pipelineRunName string) error {
			return actions.Delete(prGroupResource, cs.Dynamic, cs.Tekton.Discovery(), pipelineRunName, p.Namespace(), metav1.DeleteOptions{})
		})
		if len(prtodelete) == 0 && opts.Keep > len(prtokeep) {
			fmt.Fprintf(s.Out, "There is/are only %d %s(s) associated for Pipeline: %s \n", len(prtokeep), opts.Resource, opts.ParentResourceName)
			return nil
		}
		d.DeleteRelated(s, []string{opts.ParentResourceName})
	}

	if !opts.DeleteAllNs {
		if d.Errors() == nil {
			switch {
			case opts.Keep > 0 && opts.KeepSince > 0 && !opts.IgnoreRunning:
				fmt.Fprintf(s.Out, "%d PipelineRuns associated with %s %q has been deleted in namespace %q\n", numberOfDeletedPr, opts.ParentResource, opts.ParentResourceName, p.Namespace())
			case opts.Keep > 0 && !opts.IgnoreRunning:
				fmt.Fprintf(s.Out, "All but %d PipelineRuns associated with Pipeline %q deleted in namespace %q\n", opts.Keep, opts.ParentResourceName, p.Namespace())
			case opts.ParentResourceName != "" && !opts.IgnoreRunning:
				fmt.Fprintf(s.Out, "All PipelineRuns associated with Pipeline %q deleted in namespace %q\n", opts.ParentResourceName, p.Namespace())
			case opts.KeepSince > 0 && !opts.IgnoreRunning:
				fmt.Fprintf(s.Out, "All but %d expired PipelineRuns associated with Pipeline %q deleted in namespace %q\n", numberOfDeletedPr, opts.ParentResourceName, p.Namespace())
			case opts.Keep > 0 && opts.KeepSince > 0:
				fmt.Fprintf(s.Out, "%d PipelineRuns(Completed) associated with %s %q has been deleted in namespace %q\n", numberOfDeletedPr, opts.ParentResource, opts.ParentResourceName, p.Namespace())
			case opts.Keep > 0:
				// Should only occur in case of --pipeline and --keep being used together
				fmt.Fprintf(s.Out, "All but %d PipelineRuns(Completed) associated with Pipeline %q deleted in namespace %q\n", opts.Keep, opts.ParentResourceName, p.Namespace())
			case opts.KeepSince > 0:
				fmt.Fprintf(s.Out, "All but %d expired PipelineRuns(Completed) associated with Pipeline %q deleted in namespace %q\n", numberOfDeletedPr, opts.ParentResourceName, p.Namespace())
			case opts.ParentResourceName != "":
				fmt.Fprintf(s.Out, "All PipelineRuns(Completed) associated with Pipeline %q deleted in namespace %q\n", opts.ParentResourceName, p.Namespace())

			default:
				d.PrintSuccesses(s)
			}
		}
	} else if opts.DeleteAllNs {
		if d.Errors() == nil {
			switch {
			case opts.Keep > 0 && opts.KeepSince > 0 && !opts.IgnoreRunning:
				fmt.Fprintf(s.Out, "%d PipelineRuns has been deleted in namespace %q, kept %d\n", numberOfDeletedPr, p.Namespace(), numberOfKeptPr)
			case opts.Keep > 0 && !opts.IgnoreRunning:
				fmt.Fprintf(s.Out, "All but %d PipelineRuns deleted in namespace %q\n", opts.Keep, p.Namespace())
			case opts.KeepSince > 0 && !opts.IgnoreRunning:
				fmt.Fprintf(s.Out, "%d expired PipelineRuns has been deleted in namespace %q, kept %d\n", numberOfDeletedPr, p.Namespace(), numberOfKeptPr)
			case opts.Keep > 0 && opts.KeepSince > 0:
				fmt.Fprintf(s.Out, "%d PipelineRuns(Completed) has been deleted in namespace %q, kept %d\n", numberOfDeletedPr, p.Namespace(), numberOfKeptPr)
			case opts.Keep > 0:
				fmt.Fprintf(s.Out, "All but %d PipelineRuns(Completed) deleted in namespace %q\n", opts.Keep, p.Namespace())
			case opts.KeepSince > 0:
				fmt.Fprintf(s.Out, "%d expired PipelineRuns(Completed) has been deleted in namespace %q, kept %d\n", numberOfDeletedPr, p.Namespace(), numberOfKeptPr)
			case !opts.IgnoreRunning:
				fmt.Fprintf(s.Out, "All PipelineRuns deleted in namespace %q\n", p.Namespace())
			default:
				fmt.Fprintf(s.Out, "All PipelineRuns(Completed) deleted in namespace %q\n", p.Namespace())
			}
		}
	}
	return d.Errors()
}

func pipelineRunLister(cs *cli.Clients, keep, since int, ns string, ignoreRunning bool) func(string) ([]string, error) {
	return func(pipelineName string) ([]string, error) {
		labelSelector := fmt.Sprintf("tekton.dev/pipeline=%s", pipelineName)
		prtodelete, _, err := allPipelineRunNames(cs, keep, since, ignoreRunning, labelSelector, ns)
		if err != nil {
			return nil, err
		}
		return prtodelete, nil
	}
}

func allPipelineRunNames(cs *cli.Clients, keep, since int, ignoreRunning bool, labelselector, ns string) ([]string, []string, error) {
	var todelete, tokeep []string

	options := metav1.ListOptions{
		LabelSelector: labelselector,
	}

	var pipelineRuns *v1.PipelineRunList
	if err := actions.ListV1(pipelineRunGroupResource, cs, options, ns, &pipelineRuns); err != nil {
		return todelete, tokeep, err
	}

	if ignoreRunning {
		var pipelineRunTmps = []v1.PipelineRun{}
		for _, v := range pipelineRuns.Items {
			if v.Status.Conditions == nil {
				continue
			}
			for _, v2 := range v.Status.Conditions {
				if v2.Status == "Unknown" || v2.Reason == "Running" || v2.Reason == "Pending" || v2.Reason == "PipelineRunPending" || v2.Reason == "Started" {
					continue
				}
				pipelineRunTmps = append(pipelineRunTmps, v)
				break
			}
		}
		pipelineRuns.Items = pipelineRunTmps
	}
	switch {
	case since > 0 && keep > 0:
		todelete, tokeep = keepPipelineRunsByAgeAndNumber(pipelineRuns, since, keep, ignoreRunning)
	case since > 0:
		todelete, tokeep = keepPipelineRunsByAge(pipelineRuns, since, ignoreRunning)
	default:
		todelete, tokeep = keepPipelineRunsByNumber(pipelineRuns, keep)
	}
	return todelete, tokeep, nil
}

func keepPipelineRunsByAge(pipelineRuns *v1.PipelineRunList, keep int, ignoreRunning bool) ([]string, []string) {
	var todelete, tokeep []string
	for _, run := range pipelineRuns.Items {
		if run.Status.Conditions == nil {
			continue
		}
		switch {
		// for PipelineRuns in running status
		case !ignoreRunning && run.Status.CompletionTime == nil:
			todelete = append(todelete, run.Name)
		case time.Since(run.Status.CompletionTime.Time) > time.Duration(keep)*time.Minute:
			todelete = append(todelete, run.Name)
		default:
			tokeep = append(tokeep, run.Name)
		}
	}
	return todelete, tokeep
}

func keepPipelineRunsByNumber(pipelineRuns *v1.PipelineRunList, keep int) ([]string, []string) {
	var todelete, tokeep []string
	counter := 0

	// Do not sort PipelineRuns if keep=0 since ordering won't matter
	if keep > 0 {
		prsort.SortByStartTime(pipelineRuns.Items)
	}

	for _, run := range pipelineRuns.Items {
		if keep > 0 && counter != keep {
			counter++
			tokeep = append(tokeep, run.Name)
			continue
		}
		todelete = append(todelete, run.Name)
	}
	return todelete, tokeep
}

func keepPipelineRunsByAgeAndNumber(pipelineRuns *v1.PipelineRunList, since int, keep int, ignoreRunning bool) ([]string, []string) {
	var todelete, tokeep []string

	todelete, tokeep = keepPipelineRunsByAge(pipelineRuns, since, ignoreRunning)

	if len(tokeep) != keep {
		todelete, tokeep = keepPipelineRunsByNumber(pipelineRuns, keep)
	}
	return todelete, tokeep
}
