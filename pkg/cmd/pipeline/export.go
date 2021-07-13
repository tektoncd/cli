package pipeline

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/export"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/pipeline"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

type ExportOptions struct {
	Params       cli.Params
	PipelineName string
}

func exportCommand(p cli.Params) *cobra.Command {
	f := cliopts.NewPrintFlags("export")

	eg := `List all PipelineRuns of Pipeline 'foo':

    tkn pipelinerun list foo -n bar

List all PipelineRuns in a namespace 'foo':

    tkn pr list -n foo
`

	c := &cobra.Command{
		Use:     "export",
		Short:   "Export Pipeline",
		Example: eg,
		Annotations: map[string]string{
			"commandType": "main",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &options.DescribeOptions{Params: p}
			if len(args) == 0 {
				pipelineNames, err := pipeline.GetAllPipelineNames(p)
				if err != nil {
					return err
				}
				if len(pipelineNames) == 1 {
					opts.PipelineName = pipelineNames[0]
				} else {
					err = askPipelineName(opts, pipelineNames)
					if err != nil {
						return err
					}
				}
			} else {
				opts.PipelineName = args[0]
			}

			exported, err := exportPipeline(cmd.OutOrStdout(), p, opts.PipelineName)
			if err != nil {
				return err
			}
			fmt.Println(exported)
			return nil
		},
	}
	f.AddFlags(c)
	return c
}

func exportPipeline(out io.Writer, p cli.Params, pname string) (string, error) {
	cs, err := p.Clients()
	if err != nil {
		return "", err
	}

	pipeline, err := pipeline.Get(cs, pname, metav1.GetOptions{}, p.Namespace())
	if err != nil {
		return "", err
	}

	return export.PipelineToYaml(pipeline)
}
