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

package pipeline

import (
	"io"
	"text/tabwriter"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/pipeline"
	validate "github.com/tektoncd/cli/pkg/validate"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const graphTemplate = `digraph {
  graph [rankdir=TB];
  node [shape=box];
  "<start>" [label=start,shape=diamond];
{{- range $i, $t := .Pipeline.Spec.Tasks }}
  "{{ $t.Name }}";
{{- range $t.RunAfter }}
  "{{ . }}" -> "{{ $t.Name }}";
{{- else }}
  "<start>" -> "{{ $t.Name }}";
{{- end }}
{{- end }}
}`

func graphCommand(p cli.Params) *cobra.Command {
	c := &cobra.Command{
		Use:     "graph",
		Aliases: []string{"g"},
		Short:   "Return a graph repesentation of pipeline in a namespace",
		Annotations: map[string]string{
			"commandType": "main",
		},
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if err := validate.NamespaceExists(p); err != nil {
				return err
			}

			return printPipelineGraph(cmd.OutOrStdout(), p, args[0])
		},
	}

	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_pipeline")
	return c
}

func printPipelineGraph(out io.Writer, p cli.Params, pname string) error {
	cs, err := p.Clients()
	if err != nil {
		return err
	}

	pipeline, err := pipeline.Get(cs, pname, metav1.GetOptions{}, p.Namespace())
	if err != nil {
		return err
	}

	if len(pipeline.Spec.Resources) > 0 {
		pipeline.Spec.Resources = sortResourcesByTypeAndName(pipeline.Spec.Resources)
	}

	var data = struct {
		Pipeline     *v1beta1.Pipeline
		PipelineName string
		Params       cli.Params
	}{
		Pipeline:     pipeline,
		PipelineName: pname,
		Params:       p,
	}

	w := tabwriter.NewWriter(out, 0, 5, 3, ' ', tabwriter.TabIndent)
	t := template.Must(template.New("Pipeline Graph").Parse(graphTemplate))
	err = t.Execute(w, data)
	if err != nil {
		return err
	}

	return w.Flush()
}
