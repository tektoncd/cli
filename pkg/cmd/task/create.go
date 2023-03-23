// Copyright © 2021 The Tekton Authors.
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
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/clustertask"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/task"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type createOptions struct {
	From string
}

func createCommand(p cli.Params) *cobra.Command {
	opts := &createOptions{}
	eg := `Create a Task from ClusterTask 'foo' in namespace 'ns':
	tkn task create --from foo
or
	tkn task create foobar --from=foo -n ns`

	c := &cobra.Command{
		Use:               "create",
		ValidArgsFunction: formatted.ParentCompletion,
		Short:             "Create a Task from ClusterTask",
		Example:           eg,
		Annotations: map[string]string{
			"commandType": "main",
		},
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}
			if opts.From == "" {
				return errors.New("--from flag not passed")
			}

			if len(args) != 0 {
				return createTaskFromClusterTask(s, p, args[0], opts.From)
			}
			return createTaskFromClusterTask(s, p, "", opts.From)

		},
	}
	c.Flags().StringVarP(&opts.From, "from", "", "", "Create a ClusterTask from Task in a particular namespace")
	c.Deprecated = "ClusterTasks are deprecated, this command will be removed in future releases."
	return c
}

func createTaskFromClusterTask(s *cli.Stream, p cli.Params, tname, ctname string) error {

	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	if tname == "" {
		tname = ctname
	}

	ns := p.Namespace()

	t, _ := task.Get(cs, tname, metav1.GetOptions{}, ns)

	if t != nil {
		return fmt.Errorf(errTaskAlreadyPresent, tname, ns)
	}

	ct, err := clustertask.Get(cs, ctname, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("ClusterTask %s does not exist", ctname)
	}

	newT, err := task.Create(cs, clustertaskToTask(ct, tname), metav1.CreateOptions{}, p.Namespace())
	if err != nil {
		return err
	}

	fmt.Fprintf(s.Out, "Task %s created from ClusterTask %s in namespace %s\n", newT.Name, ct.Name, p.Namespace())

	return nil
}

func clustertaskToTask(ct *v1beta1.ClusterTask, tName string) *v1beta1.Task {
	t := &v1beta1.Task{}

	// Copy required Metadata from Task to ClusterTask
	t.ObjectMeta = metav1.ObjectMeta{
		Name:            tName,
		Labels:          ct.Labels,
		Annotations:     ct.Annotations,
		OwnerReferences: ct.OwnerReferences,
	}
	t.TypeMeta = metav1.TypeMeta{
		APIVersion: ct.APIVersion,
		Kind:       "Task",
	}
	// Copy the Specs from ClusterTask to Task
	t.Spec = ct.Spec

	return t
}
