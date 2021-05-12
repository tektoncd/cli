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

package search

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/tektoncd/hub/api/pkg/cli/app"
	"github.com/tektoncd/hub/api/pkg/cli/flag"
	"github.com/tektoncd/hub/api/pkg/cli/formatter"
	"github.com/tektoncd/hub/api/pkg/cli/hub"
	"github.com/tektoncd/hub/api/pkg/cli/printer"
	"github.com/tektoncd/hub/api/pkg/parser"
)

const resTemplate = `{{- $rl := len .Resources }}{{ if eq $rl 0 -}}
No Resources found
{{ else -}}
NAME	KIND	CATALOG	DESCRIPTION	TAGS
{{ range $_, $r := .Resources -}}
{{ formatName $r.Name $r.LatestVersion.Version }}	{{ $r.Kind }}	{{ formatCatalogName $r.Catalog.Name }}	{{ formatDesc $r.LatestVersion.Description 40 }}	{{ formatTags $r.Tags }}
{{ end }}
{{- end -}}
`

var (
	funcMap = template.FuncMap{
		"formatName":        formatter.FormatName,
		"formatCatalogName": formatter.FormatCatalogName,
		"formatDesc":        formatter.FormatDesc,
		"formatTags":        formatter.FormatTags,
	}
	tmpl = template.Must(template.New("List Resources").Funcs(funcMap).Parse(resTemplate))
)

type options struct {
	cli    app.CLI
	limit  uint
	match  string
	output string
	tags   []string
	kinds  []string
	args   []string
}

var examples string = `
Search a resource of name 'foo':

    tkn hub search foo

or

Search resources using tag 'cli':

    tkn hub search --tags cli
`

func Command(cli app.CLI) *cobra.Command {

	opts := &options{cli: cli}

	cmd := &cobra.Command{
		Use:     "search",
		Short:   "Search resource by a combination of name, kind, and tags",
		Long:    ``,
		Example: examples,
		Annotations: map[string]string{
			"commandType": "main",
		},
		SilenceUsage: true,
		Args:         cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.args = args
			return opts.run()
		},
	}

	cmd.Flags().UintVarP(&opts.limit, "limit", "l", 0, "Max number of resources to fetch")
	cmd.Flags().StringVar(&opts.match, "match", "contains", "Accept type of search. 'exact' or 'contains'.")
	cmd.Flags().StringArrayVar(&opts.kinds, "kinds", nil, "Accepts a comma separated list of kinds")
	cmd.Flags().StringArrayVar(&opts.tags, "tags", nil, "Accepts a comma separated list of tags")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "table", "Accepts output format: [table, json]")

	return cmd
}

func (opts *options) run() error {

	if err := opts.validate(); err != nil {
		return err
	}

	hubClient := opts.cli.Hub()

	result := hubClient.Search(hub.SearchOption{
		Name:  opts.name(),
		Kinds: opts.kinds,
		Tags:  opts.tags,
		Match: opts.match,
		Limit: opts.limit,
	})

	out := opts.cli.Stream().Out

	if opts.output == "json" {
		return printer.New(out).JSON(result.Raw())
	}

	typed, err := result.Typed()
	if err != nil {
		return err
	}

	var templateData = struct {
		Resources hub.SearchResponse
	}{
		Resources: typed,
	}

	return printer.New(out).Tabbed(tmpl, templateData)
}

func (opts *options) validate() error {

	if flag.AllEmpty(opts.args, opts.kinds, opts.tags) {
		return fmt.Errorf("please specify a resource name, --tags or --kinds flag to search")
	}

	if err := flag.InList("match", opts.match, []string{"contains", "exact"}); err != nil {
		return err
	}

	if err := flag.InList("output", opts.output, []string{"table", "json"}); err != nil {
		return err
	}

	opts.kinds = flag.TrimArray(opts.kinds)
	opts.tags = flag.TrimArray(opts.tags)

	for _, k := range opts.kinds {
		if !parser.IsSupportedKind(k) {
			return fmt.Errorf("invalid value %q set for option kinds. supported kinds: [%s]",
				k, strings.ToLower(strings.Join(parser.SupportedKinds(), ", ")))
		}
	}
	return nil
}

func (opts *options) name() string {
	if len(opts.args) == 0 {
		return ""
	}
	return strings.TrimSpace(opts.args[0])
}
