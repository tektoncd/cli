// Copyright © 2020 The Tekton Authors.
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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/flags"
	"github.com/tektoncd/cli/pkg/taskrun"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type restartOptions struct {
	PrefixName    string
	TektonOptions flags.TektonOptions
}

func restartCommand(p cli.Params) *cobra.Command {
	eg := `Restart a TaskRun named foo in namespace bar:

	tkn taskrun restart foo -n bar
`

	opt := restartOptions{}
	c := &cobra.Command{
		Use:          "restart",
		Short:        "Restart a TaskRun in a namespace",
		Example:      eg,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			opt.TektonOptions = flags.GetTektonOptions(cmd)
			return restartTaskRun(p, s, args[0], opt)
		},
	}

	c.Flags().StringVar(&opt.PrefixName, "prefix-name", "", "Specify a prefix for the TaskRun name (must be lowercase alphanumeric characters)")
	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_taskrun")
	return c
}

func restartTaskRun(p cli.Params, s *cli.Stream, trName string, opt restartOptions) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	tr, err := taskrun.GetV1beta1(cs, trName, metav1.GetOptions{}, p.Namespace())
	if err != nil {
		return fmt.Errorf("failed to find TaskRun: %s", trName)
	}

	var generateName string
	if opt.PrefixName != "" {
		generateName = opt.PrefixName + "-"
	} else {
		if tr.GenerateName == "" {
			return fmt.Errorf("generateName field required for TaskRun with restart\nUse --prefix-name option if TaskRun does not have generateName set")
		}
		generateName = tr.GenerateName
	}

	newTr := v1beta1.TaskRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "TaskRun",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    p.Namespace(),
			GenerateName: generateName,
		},
		Spec: tr.Spec,
	}
	// Reapply blank status in case TaskRun was cancelled
	newTr.Spec.Status = ""

	trCreated, err := taskrun.Create(cs, &newTr, metav1.CreateOptions{}, p.Namespace())
	if err != nil {
		return err
	}

	fmt.Fprintf(s.Out, "TaskRun started: %s\n", trCreated.Name)
	inOrderString := "\nIn order to track the TaskRun progress run:\ntkn taskrun "
	if opt.TektonOptions.Context != "" {
		inOrderString += fmt.Sprintf("--context=%s ", opt.TektonOptions.Context)
	}
	inOrderString += fmt.Sprintf("logs %s -f -n %s\n", trCreated.Name, trCreated.Namespace)
	fmt.Fprint(s.Out, inOrderString)

	return nil
}
