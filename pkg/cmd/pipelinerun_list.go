// Copyright © 2019 The Knative Authors.
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

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cli "k8s.io/cli-runtime/pkg/genericclioptions"
)

// pipelinerunListCmd holds everything related to "pipeline runs list" to
// namespace all package variables
var pipelinerunListCmd = struct {
	cmd      *cobra.Command
	cliFlags *cli.PrintFlags
}{}

func init() {
	pipelinerunCmd.AddCommand(pipelinerunList())
}

func pipelinerunList() *cobra.Command {
	if pipelinerunListCmd.cmd != nil {
		return pipelinerunListCmd.cmd
	}

	c := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "A brief description of your command",
		Long: `A longer description that spans multiple lines and likely contains examples
	and usage of using your command. For example:

	Cobra is a CLI library for Go that empowers applications.
	This application is a tool to generate the needed files
	to quickly create a Cobra application.`,
		RunE: getpipelineruns,
	}

	defaultOutput := `jsonpath={range .items[*]}{.metadata.name}{"\n"}{end}`
	f := cli.NewPrintFlags("").WithDefaultOutput(defaultOutput)
	f.AddFlags(c)

	pipelinerunListCmd.cmd = c
	pipelinerunListCmd.cliFlags = f

	return pipelinerunListCmd.cmd
}

func getpipelineruns(cmd *cobra.Command, args []string) error {
	var err error

	cs, err := versioned.NewForConfig(kubeConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create client from config %s  %s", kubeCfgFile, err)
		return err
	}
	c := cs.TektonV1alpha1().PipelineRuns(namespace)
	ps, err := c.List(v1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to list pipelinerun from namespace %s  %s", namespace, err)
		return err
	}
	printer, err := pipelinerunListCmd.cliFlags.ToPrinter()
	if err != nil {
		return err
	}
	return printer.PrintObj(ps, cmd.OutOrStdout())
}
