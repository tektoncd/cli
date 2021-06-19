package pipeline

import (
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

	eg := `Export Pipeline Definition:

	tkn pipeline export will export a pipeline definition as yaml to be easily
	reimported or modified.

	Example: export a Pipeline named 'pipeline' in namespace 'foo' and recreate
	it in the namespace 'bar':

    tkn p export pipeline -n foo|kubectl create -f- -n bar
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

			err := exportPipeline(cmd.OutOrStdout(), p, opts.PipelineName)
			if err != nil {
				return err
			}
			return nil
		},
	}
	f.AddFlags(c)
	return c
}

func exportPipeline(out io.Writer, p cli.Params, pname string) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	pipeline, err := pipeline.Get(cs, pname, metav1.GetOptions{}, p.Namespace())
	if err != nil {
		return err
	}
	exported, err := export.PipelineToYaml(pipeline)
	if err != nil {
		return err
	}
	_, err = out.Write([]byte(exported))
	return err
}
