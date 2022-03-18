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
	"bufio"
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
	"github.com/tektoncd/cli/pkg/options"
)

func watchCommand(p cli.Params) *cobra.Command {
	showLog := false

	opts := &options.LogOptions{Params: p}
	eg := `watch a PipelineRun of name 'foo' in namespace 'bar':

    tkn pipelinerun watch foo -n bar

this command will wait for the pipelinerun to finish and exit with the exit status of the
pipelinerun.

`

	c := &cobra.Command{
		Use:           "watch",
		Short:         "Watch a PipelineRun until it finishes",
		Example:       eg,
		SilenceUsage:  true,
		SilenceErrors: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		ValidArgsFunction: formatted.ParentCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				opts.PipelineRunName = args[0]
			}

			devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0o755)
			if err != nil {
				return err
			}
			var errstreamb bytes.Buffer
			wb := bufio.NewWriter(&errstreamb)

			opts.Follow = true
			opts.Limit = defaultLimit
			opts.Stream = &cli.Stream{
				Err: wb,
			}

			if showLog {
				opts.Stream.Out = cmd.OutOrStdout()
				cmd.SilenceErrors = false
			} else {
				opts.Stream.Out = devnull
			}

			err = Run(opts)
			if err != nil {
				return err
			}
			err = wb.Flush()
			if err != nil {
				return err
			}
			if errstreamb.String() != "" {
				return fmt.Errorf(errstreamb.String())
			}
			return nil
		},
	}

	c.Flags().BoolVarP(&opts.Last, "last", "L", false, "watch last PipelineRun")
	c.Flags().BoolVarP(&showLog, "showlog", "", false, "show logs of the PipelineRun")
	return c
}
