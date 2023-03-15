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

package taskrun

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
	taskpkg "github.com/tektoncd/cli/pkg/task"
	trsort "github.com/tektoncd/cli/pkg/taskrun/sort"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

type deleteOptions struct {
	ClusterTaskName string
	TaskName        string
}

// trExists validates that the arguments are valid TaskRun names
func trExists(args []string, p cli.Params) ([]string, error) {

	availableTrs := make([]string, 0)
	c, err := p.Clients()
	if err != nil {
		return availableTrs, err
	}
	var errorList error
	ns := p.Namespace()
	for _, name := range args {
		var tr *v1.TaskRun
		if err = actions.GetV1(taskrunGroupResource, c, name, ns, metav1.GetOptions{}, &tr); err != nil {
			errorList = multierr.Append(errorList, err)
			continue
		}

		availableTrs = append(availableTrs, name)
	}
	return availableTrs, errorList
}

func deleteCommand(p cli.Params) *cobra.Command {
	opts := &options.DeleteOptions{Resource: "TaskRun", ForceDelete: false, DeleteAllNs: false}
	deleteOpts := &deleteOptions{}
	f := cliopts.NewPrintFlags("delete")
	eg := `Delete TaskRuns with names 'foo' and 'bar' in namespace 'quux':

    tkn taskrun delete foo bar -n quux

or

    tkn tr rm foo bar -n quux
`

	c := &cobra.Command{
		Use:               "delete",
		Aliases:           []string{"rm"},
		Short:             "Delete TaskRuns in a namespace",
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

			if deleteOpts.TaskName != "" && deleteOpts.ClusterTaskName != "" {
				return fmt.Errorf("cannot use --task and --clustertask option together")
			}

			if deleteOpts.ClusterTaskName != "" {
				opts.ParentResource = "ClusterTask"
				opts.ParentResourceName = deleteOpts.ClusterTaskName
			} else {
				opts.ParentResource = "Task"
				opts.ParentResourceName = deleteOpts.TaskName
			}

			if opts.Keep < 0 && opts.KeepSince < 0 {
				return fmt.Errorf("keep and keep-since option should not be lower than 0")
			}

			if opts.Keep < 0 {
				return fmt.Errorf("keep option should not be lower than 0")
			}

			if opts.KeepSince < 0 {
				return fmt.Errorf("since option should not be lower than 0")
			}

			if (opts.Keep > 0 || opts.KeepSince > 0) && opts.ParentResourceName == "" {
				opts.DeleteAllNs = true
			}

			if (opts.Keep > 0 || opts.KeepSince > 0) && opts.DeleteAllNs && opts.ParentResourceName != "" {
				return fmt.Errorf("--keep or --keep-since, --all and --%s cannot be used together", strings.ToLower(opts.ParentResource))
			}

			availableTrs, errs := trExists(args, p)
			if len(availableTrs) == 0 && errs != nil {
				return errs
			}

			if err := opts.CheckOptions(s, availableTrs, p.Namespace()); err != nil {
				return err
			}

			if err := deleteTaskRuns(s, p, availableTrs, opts); err != nil {
				return err
			}
			return errs
		},
	}
	f.AddFlags(c)
	c.Flags().BoolVarP(&opts.ForceDelete, "force", "f", false, "Whether to force deletion (default: false)")
	c.Flags().StringVarP(&deleteOpts.TaskName, "task", "t", "", "The name of a Task whose TaskRuns should be deleted (does not delete the task)")
	c.Flags().StringVarP(&deleteOpts.ClusterTaskName, "clustertask", "", "", "The name of a ClusterTask whose TaskRuns should be deleted (does not delete the ClusterTask)")
	c.Flags().BoolVarP(&opts.DeleteAllNs, "all", "", false, "Delete all TaskRuns in a namespace (default: false)")
	c.Flags().IntVarP(&opts.Keep, "keep", "", 0, "Keep n most recent number of TaskRuns")
	c.Flags().IntVarP(&opts.KeepSince, "keep-since", "", 0, "When deleting all TaskRuns keep the ones that has been completed since n minutes")
	c.Flags().BoolVarP(&opts.IgnoreRunning, "ignore-running", "i", true, "ignore running TaskRun (default: true)")
	c.Flags().BoolVarP(&opts.IgnoreRunningPipelinerun, "ignore-running-pipelinerun", "", true, "ignore deleting taskruns of a running PipelineRun (default: true)")

	return c
}

func deleteTaskRuns(s *cli.Stream, p cli.Params, trNames []string, opts *options.DeleteOptions) error {
	var numberOfDeletedTr, numberOfKeptTr int
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}
	var d *deleter.Deleter
	switch {
	case opts.DeleteAllNs:
		d = deleter.New("TaskRun", func(taskRunName string) error {
			return actions.Delete(taskrunGroupResource, cs.Dynamic, cs.Tekton.Discovery(), taskRunName, p.Namespace(), metav1.DeleteOptions{})
		})
		trToDelete, trToKeep, err := allTaskRunNames(cs, opts.Keep, opts.KeepSince, opts.IgnoreRunning, opts.IgnoreRunningPipelinerun, opts.LabelSelector, p.Namespace(), "")
		if err != nil {
			return err
		}
		numberOfDeletedTr = len(trToDelete)
		numberOfKeptTr = len(trToKeep)
		d.Delete(s, trToDelete)
	case opts.ParentResourceName == "":
		d = deleter.New("TaskRun", func(taskRunName string) error {
			return actions.Delete(taskrunGroupResource, cs.Dynamic, cs.Tekton.Discovery(), taskRunName, p.Namespace(), metav1.DeleteOptions{})
		})
		var processedTrNames []string

		for _, trNane := range trNames {
			var tr *v1.TaskRun
			if err = actions.GetV1(taskrunGroupResource, cs, trNane, p.Namespace(), metav1.GetOptions{}, &tr); err != nil {
				return fmt.Errorf("failed to get taskrun")
			}
			prFinished := ownerPrFinished(cs, *tr)

			if !prFinished && opts.ForceDelete {
				fmt.Fprintf(s.Out, "warning: Taskrun %s related pipelinerun still running.\n", tr.Name)
			}
			if !prFinished && !opts.ForceDelete {
				fmt.Fprintf(s.Out, "TaskRun(s): %s attached to PipelineRun is still running deleting will restart the completed taskrun. Proceed (y/n): ", tr.Name)
				if err := opts.TakeInput(s, ""); err != nil {
					continue
				}
			}
			processedTrNames = append(processedTrNames, tr.Name)
		}
		d.Delete(s, processedTrNames)
	default:
		d = deleter.New(opts.ParentResource, func(_ string) error {
			err := fmt.Sprintf("the %s should not be deleted", opts.ParentResource)
			return errors.New(err)
		})

		// Create a LabelSelector to filter the TaskRuns which are associated with particular
		// Task or ClusterTask
		resourceType := "task"
		if opts.ParentResource == "ClusterTask" {
			resourceType = "clusterTask"
		}
		labelSelector := fmt.Sprintf("tekton.dev/%s=%s", resourceType, opts.ParentResourceName)

		// Compute the total no of TaskRuns which we need to delete
		trToDelete, trToKeep, err := allTaskRunNames(cs, opts.Keep, opts.KeepSince, opts.IgnoreRunning, opts.IgnoreRunningPipelinerun, labelSelector, p.Namespace(), opts.ParentResource)
		if err != nil {
			return err
		}
		numberOfDeletedTr = len(trToDelete)
		numberOfKeptTr = len(trToKeep)

		// Delete the TaskRuns associated with a Task or ClusterTask
		d.WithRelated("TaskRun", taskRunLister(p, opts.Keep, opts.KeepSince, opts.ParentResource, cs, opts.IgnoreRunning, opts.IgnoreRunningPipelinerun), func(taskRunName string) error {
			return actions.Delete(taskrunGroupResource, cs.Dynamic, cs.Tekton.Discovery(), taskRunName, p.Namespace(), metav1.DeleteOptions{})
		})
		if len(trToDelete) == 0 && opts.Keep > len(trToKeep) {
			fmt.Fprintf(s.Out, "There is/are only %d %s(s) associated for %s: %s \n", len(trToKeep), opts.Resource, opts.ParentResource, opts.ParentResourceName)
			return nil
		}
		d.DeleteRelated(s, []string{opts.ParentResourceName})
	}

	if !opts.DeleteAllNs {
		if d.Errors() == nil {
			switch {
			case opts.Keep > 0 && opts.KeepSince > 0 && !opts.IgnoreRunning:
				fmt.Fprintf(s.Out, "%d TaskRuns associated with %s %q has been deleted in namespace %q\n", numberOfDeletedTr, opts.ParentResource, opts.ParentResourceName, p.Namespace())
			case opts.Keep > 0 && !opts.IgnoreRunning:
				// Should only occur in case of --task flag and --keep being used together
				fmt.Fprintf(s.Out, "All but %d TaskRuns associated with %s %q deleted in namespace %q\n", opts.Keep, opts.ParentResource, opts.ParentResourceName, p.Namespace())
			case opts.KeepSince > 0 && !opts.IgnoreRunning:
				fmt.Fprintf(s.Out, "All but %d expired TaskRuns associated with %q %q deleted in namespace %q\n", numberOfDeletedTr, opts.ParentResource, opts.ParentResourceName, p.Namespace())
			case opts.ParentResourceName != "" && !opts.IgnoreRunning:
				fmt.Fprintf(s.Out, "All %d TaskRuns associated with %s %q deleted in namespace %q\n", numberOfDeletedTr, opts.ParentResource, opts.ParentResourceName, p.Namespace())
			case opts.Keep > 0 && opts.KeepSince > 0:
				fmt.Fprintf(s.Out, "%d TaskRuns(Completed) associated with %s %q has been deleted in namespace %q\n", numberOfDeletedTr, opts.ParentResource, opts.ParentResourceName, p.Namespace())
			case opts.Keep > 0:
				// Should only occur in case of --task flag and --keep being used together
				fmt.Fprintf(s.Out, "All but %d TaskRuns(Completed) associated with %s %q deleted in namespace %q\n", opts.Keep, opts.ParentResource, opts.ParentResourceName, p.Namespace())
			case opts.KeepSince > 0:
				fmt.Fprintf(s.Out, "All but %d expired TaskRuns associated with %q %q deleted in namespace %q\n", numberOfDeletedTr, opts.ParentResource, opts.ParentResourceName, p.Namespace())
			case opts.ParentResourceName != "":
				fmt.Fprintf(s.Out, "All %d TaskRuns(Completed) associated with %s %q deleted in namespace %q\n", numberOfDeletedTr, opts.ParentResource, opts.ParentResourceName, p.Namespace())
			default:
				d.PrintSuccesses(s)
			}
		}
	} else if opts.DeleteAllNs {
		if d.Errors() == nil {
			switch {
			case opts.Keep > 0 && opts.KeepSince > 0 && !opts.IgnoreRunning:
				fmt.Fprintf(s.Out, "%d TaskRuns has been deleted in namespace %q, kept %d\n", numberOfDeletedTr, p.Namespace(), numberOfKeptTr)
			case opts.Keep > 0 && !opts.IgnoreRunning:
				fmt.Fprintf(s.Out, "All but %d TaskRuns deleted in namespace %q\n", opts.Keep, p.Namespace())
			case opts.KeepSince > 0 && !opts.IgnoreRunning:
				fmt.Fprintf(s.Out, "%d expired Taskruns has been deleted in namespace %q, kept %d\n", numberOfDeletedTr, p.Namespace(), numberOfKeptTr)
			case opts.Keep > 0 && opts.KeepSince > 0:
				fmt.Fprintf(s.Out, "%d TaskRuns(Completed) has been deleted in namespace %q, kept %d\n", numberOfDeletedTr, p.Namespace(), numberOfKeptTr)
			case opts.Keep > 0:
				fmt.Fprintf(s.Out, "All but %d TaskRuns(Completed) deleted in namespace %q\n", opts.Keep, p.Namespace())
			case opts.KeepSince > 0:
				fmt.Fprintf(s.Out, "%d expired Taskruns(Completed) has been deleted in namespace %q, kept %d\n", numberOfDeletedTr, p.Namespace(), numberOfKeptTr)
			case !opts.IgnoreRunning:
				fmt.Fprintf(s.Out, "All %d TaskRuns deleted in namespace %q\n", numberOfDeletedTr, p.Namespace())
			default:
				fmt.Fprintf(s.Out, "All %d TaskRuns(Completed) deleted in namespace %q\n", numberOfDeletedTr, p.Namespace())
			}
		}
	}
	return d.Errors()
}

func taskRunLister(p cli.Params, keep, since int, kind string, cs *cli.Clients, ignoreRunning, ignoreRunningOwner bool) func(string) ([]string, error) {
	return func(taskName string) ([]string, error) {
		label := "task"
		if kind == "ClusterTask" {
			label = "clusterTask"
		}

		labelSelector := fmt.Sprintf("tekton.dev/%s=%s", label, taskName)
		trToDelete, _, err := allTaskRunNames(cs, keep, since, ignoreRunning, ignoreRunningOwner, labelSelector, p.Namespace(), kind)
		if err != nil {
			return nil, err
		}
		return trToDelete, nil
	}
}

func allTaskRunNames(cs *cli.Clients, keep, since int, ignoreRunning, ignoreRunningOwner bool, labelSelector, ns, kind string) ([]string, []string, error) {
	var todelete, tokeep []string
	options := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	var taskRuns *v1.TaskRunList
	if err := actions.ListV1(taskrunGroupResource, cs, options, ns, &taskRuns); err != nil {
		return todelete, tokeep, err
	}

	if kind == "Task" {
		taskRuns.Items = taskpkg.FilterByRef(taskRuns.Items, string(v1.NamespacedTaskKind))
	}

	if ignoreRunningOwner {
		taskRuns = trsWithOwnerPrFinished(cs, taskRuns)
	}

	if ignoreRunning {
		var taskRunTmp = []v1.TaskRun{}
		for _, v := range taskRuns.Items {

			if v.Status.CompletionTime == nil {
				// Skip TaskRuns without CompletionTimes as they have not finished running yet
				continue
			}
			taskRunTmp = append(taskRunTmp, v)
		}
		taskRuns.Items = taskRunTmp
	}
	switch {
	case since > 0 && keep > 0:
		todelete, tokeep = keepTaskRunsByAgeAndNumber(taskRuns, since, keep, ignoreRunning)
	case since > 0:
		todelete, tokeep = keepTaskRunsByAge(taskRuns, since, ignoreRunning)
	default:
		todelete, tokeep = keepTaskRunsByNumber(taskRuns, keep)
	}
	return todelete, tokeep, nil
}

func keepTaskRunsByAge(taskRuns *v1.TaskRunList, since int, ignoreRunning bool) ([]string, []string) {
	var todelete, tokeep []string

	for _, run := range taskRuns.Items {
		switch {
		// for TaskRuns in running status
		case !ignoreRunning && run.Status.CompletionTime == nil:
			todelete = append(todelete, run.Name)
		case time.Since(run.Status.CompletionTime.Time) > time.Duration(since)*time.Minute:
			todelete = append(todelete, run.Name)
		default:
			tokeep = append(tokeep, run.Name)
		}
	}
	return todelete, tokeep
}

func keepTaskRunsByNumber(taskRuns *v1.TaskRunList, keep int) ([]string, []string) {
	var todelete, tokeep []string
	var counter = 0

	// Do not sort TaskRuns if keep=0 since ordering won't matter
	if keep > 0 {
		trsort.SortByStartTime(taskRuns.Items)
	}

	for _, tr := range taskRuns.Items {
		if keep > 0 && counter != keep {
			counter++
			tokeep = append(tokeep, tr.Name)
			continue
		}
		todelete = append(todelete, tr.Name)
	}
	return todelete, tokeep
}

func keepTaskRunsByAgeAndNumber(taskRuns *v1.TaskRunList, since int, keep int, ignoreRunning bool) ([]string, []string) {
	var todelete, tokeep []string

	todelete, tokeep = keepTaskRunsByAge(taskRuns, since, ignoreRunning)

	if len(tokeep) != keep {
		todelete, tokeep = keepTaskRunsByNumber(taskRuns, keep)
	}
	return todelete, tokeep
}

func trsWithOwnerPrFinished(cs *cli.Clients, runs *v1.TaskRunList) *v1.TaskRunList {
	var filteredTrs v1.TaskRunList
	for _, tr := range runs.Items {
		if ownerPrFinished(cs, tr) {
			filteredTrs.Items = append(filteredTrs.Items, tr)
		}
	}
	return &filteredTrs
}

func ownerPrFinished(cs *cli.Clients, tr v1.TaskRun) bool {
	for _, ref := range tr.GetOwnerReferences() {
		if ref.Kind == pipeline.PipelineRunControllerName {
			var pr *v1.PipelineRun
			err := actions.GetV1(pipelineRunGroupResource, cs, tr.Namespace, ref.Name, metav1.GetOptions{}, &pr)
			if err != nil {
				return false
			}
			// return false without CompletionTimes as owner pipelinerun have not finished running yet
			return pr.Status.CompletionTime != nil
		}
	}
	return true
}
