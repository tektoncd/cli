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

package clustertask

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	clustertaskpkg "github.com/tektoncd/cli/pkg/clustertask"
	"github.com/tektoncd/cli/pkg/deleter"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

// ctExists validates that the arguments are valid ClusterTask names
func ctExists(args []string, p cli.Params) ([]string, error) {

	availableCts := make([]string, 0)
	c, err := p.Clients()
	if err != nil {
		return availableCts, err
	}
	var errorList error
	for _, name := range args {
		var clustertask *v1beta1.ClusterTask
		err := actions.GetV1(clustertaskGroupResource, c, name, "", metav1.GetOptions{}, &clustertask)
		if err != nil {
			errorList = multierr.Append(errorList, err)
			continue
		}
		availableCts = append(availableCts, name)
	}
	return availableCts, errorList
}

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "ClusterTask", ForceDelete: false, DeleteAll: false}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete ClusterTasks with names 'foo' and 'bar':

    tkn clustertask delete foo bar

or

    tkn ct rm foo bar
`

	c := &cobra.Command{
		Use:          "delete",
		Aliases:      []string{"rm"},
		Short:        "Delete ClusterTasks in a cluster",
		Example:      eg,
		Args:         cobra.MinimumNArgs(0),
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		ValidArgsFunction: formatted.ParentCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				In:  cmd.InOrStdin(),
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			availableCts, errs := ctExists(args, p)
			if len(availableCts) == 0 && errs != nil {
				return errs
			}

			if err := opts.CheckOptions(s, availableCts, ""); err != nil {
				return err
			}

			if err := deleteClusterTasks(opts, s, p, availableCts); err != nil {
				return err
			}
			return errs
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().BoolVarP(&opts.DeleteAll, "all", "", false, "Delete all ClusterTasks (default: false)")
	c.Flags().BoolVarP(&opts.DeleteRelated, "trs", "", false, "Whether to delete ClusterTask(s) and related resources (TaskRuns) (default: false)")
	c.Deprecated = "ClusterTasks are deprecated, this command will be removed in future releases."
	return c
}
func deleteClusterTasks(opts *options.DeleteOptions, s *cli.Stream, p cli.Params, ctNames []string) error {
	ctGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "clustertasks"}
	trGroupResource := schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}

	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("Failed to create tekton client")
	}
	d := deleter.New("ClusterTask", func(taskName string) error {
		return actions.Delete(ctGroupResource, cs.Dynamic, cs.Tekton.Discovery(), taskName, "", metav1.DeleteOptions{})
	})
	switch {
	case opts.DeleteAll:
		cts, err := clustertaskpkg.GetAllClusterTaskNames(clustertaskGroupResource, cs)
		if err != nil {
			return err
		}
		d.Delete(cts)
	case opts.DeleteRelated:
		d.WithRelated("TaskRun", taskRunLister(cs, p), func(taskRunName string) error {
			return actions.Delete(trGroupResource, cs.Dynamic, cs.Tekton.Discovery(), taskRunName, p.Namespace(), metav1.DeleteOptions{})
		})
		deletedClusterTaskNames := d.Delete(ctNames)
		d.DeleteRelated(deletedClusterTaskNames)
	default:
		d.Delete(ctNames)

	}

	if !opts.DeleteAll {
		d.PrintSuccesses(s)
	} else if opts.DeleteAll {
		if d.Errors() == nil {
			fmt.Fprint(s.Out, "All ClusterTasks deleted\n")
		}
	}
	return d.Errors()
}

func taskRunLister(cs *cli.Clients, p cli.Params) func(string) ([]string, error) {
	return func(taskName string) ([]string, error) {
		lOpts := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("tekton.dev/clusterTask=%s", taskName),
		}

		var taskRuns *v1.TaskRunList
		if err := actions.ListV1(taskrunGroupResource, cs, lOpts, p.Namespace(), &taskRuns); err != nil {
			return nil, err
		}
		var names []string
		for _, tr := range taskRuns.Items {
			names = append(names, tr.Name)
		}
		return names, nil
	}
}
