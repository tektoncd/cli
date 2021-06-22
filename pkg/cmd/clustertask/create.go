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

package clustertask

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	ctactions "github.com/tektoncd/cli/pkg/clustertask"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/task"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createCommand(p cli.Params) *cobra.Command {
	opts := &options.CreateOptions{}
	eg := `Create a ClusterTask from Task 'foo' present in namespace 'ns':
	tkn clustertask create --from-task foo
or
	tkn clustertask create foobar --from-task=foo`

	c := &cobra.Command{
		Use:               "create",
		ValidArgsFunction: formatted.ParentCompletion,
		Short:             "Create a ClusterTask from Task",
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
			if opts.FromTask == "" {
				return errors.New("--from-task flag not passed")
			}

			if len(args) != 0 {
				return createClusterTaskFromTask(s, p, args[0], opts.FromTask)
			}
			return createClusterTaskFromTask(s, p, opts.FromTask, opts.FromTask)

		},
	}
	c.Flags().StringVarP(&opts.FromTask, "from-task", "", "", "Create a ClusterTask from Task in a particular namespace")
	return c
}

func createClusterTaskFromTask(s *cli.Stream, p cli.Params, ctName, tName string) error {
	c, err := p.Clients()
	if err != nil {
		return err
	}

	ct, _ := ctactions.Get(c, ctName, metav1.GetOptions{})

	if ct != nil {
		return fmt.Errorf(errClusterTaskAlreadyPresent, ctName)
	}

	namespace := p.Namespace()

	t, err := task.Get(c, tName, metav1.GetOptions{}, namespace)
	if err != nil {
		return fmt.Errorf("Task %s does not exist in namespace %s", tName, namespace)
	}

	newCT, err := ctactions.Create(c, taskToClusterTask(t, ctName), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	fmt.Fprintf(s.Out, "ClusterTask %s created from Task %s present in namespace %s\n", newCT.Name, t.Name, namespace)

	return nil
}

func taskToClusterTask(t *v1beta1.Task, newCTName string) *v1beta1.ClusterTask {
	ct := &v1beta1.ClusterTask{}

	// Copy required Metadata from Task to ClusterTask
	ct.ObjectMeta = metav1.ObjectMeta{
		Name:            newCTName,
		Labels:          t.Labels,
		Annotations:     t.Annotations,
		GenerateName:    t.GenerateName,
		OwnerReferences: t.OwnerReferences,
	}
	ct.TypeMeta = metav1.TypeMeta{
		APIVersion: t.APIVersion,
		Kind:       "ClusterTask",
	}
	// Copy the Specs from Task to ClusterTask
	ct.Spec = t.Spec

	return ct
}
