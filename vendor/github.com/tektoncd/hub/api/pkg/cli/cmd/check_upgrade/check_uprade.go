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

package check_upgrade

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/tektoncd/hub/api/pkg/cli/app"
	"github.com/tektoncd/hub/api/pkg/cli/formatter"
	"github.com/tektoncd/hub/api/pkg/cli/hub"
	"github.com/tektoncd/hub/api/pkg/cli/installer"
	"github.com/tektoncd/hub/api/pkg/cli/kube"
	"github.com/tektoncd/hub/api/pkg/cli/printer"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const upgradeTemplate = `{{decorate "underline bold" "Upgrades Available\n" }}

{{- if ne (len .HubResources) 0 }}
{{- if .IsPipelineVersionUnknown }}
NAME	CATALOG	CURRENT_VERSION	LATEST_VERSION
{{- else }}
NAME	CATALOG	CURRENT_VERSION	LATEST_COMPATIBLE_VERSION
{{- end }}
{{- range $el := .HubResources }}
{{ $el.Name }}	{{ $el.Catalog }}	{{ $el.CurrentVersion }}	{{ $el.LatestVersion }}
{{- end }}
{{- else }}
No {{ .Kind }} for upgrade
{{- end -}}

{{ if ne (len .NonHubResources) 0 }}
{{ decorate "underline bold" "\nSkipped Resources\n" }}
NAME
{{- range $el := .NonHubResources }}
{{ icon "bullet" }}{{ $el }}
{{- end -}}
{{- end -}}

{{ if .IsPipelineVersionUnknown }}
{{ decorate "bold" "\nWARN: Pipelines version unknown. Check your pipelines version before upgrading." }}
{{- end }}
`

var (
	funcMap = template.FuncMap{
		"icon":     formatter.Icon,
		"decorate": formatter.DecorateAttr,
	}
	tmpl = template.Must(template.New("Check Upgrade").Funcs(funcMap).Parse(upgradeTemplate))
)

type hubRes struct {
	Name           string
	Catalog        string
	CurrentVersion string
	LatestVersion  string
}

type templateData struct {
	Kind                     string
	NonHubResources          []string
	HubResources             []hubRes
	IsPipelineVersionUnknown bool
}

const (
	versionLabel = "app.kubernetes.io/version"
	hubLabel     = "hub.tekton.dev/catalog"
)

type options struct {
	cli     app.CLI
	kind    string
	version string
	args    []string
	kc      kube.Config
	cs      kube.ClientSet
}

var cmdExamples string = `
Check for Upgrades of %S installed via Tekton Hub CLI:

	tkn hub check-upgrade %s

The above command will check for upgrades of %Ss installed via Tekton Hub CLI
and will skip the %Ss which are not installed by Tekton Hub CLI.

NOTE: If Pipelines version is unknown it will show the latest version available
else it will show latest compatible version.
`

func Command(cli app.CLI) *cobra.Command {
	opts := &options{cli: cli}

	cmd := &cobra.Command{
		Use:   "check-upgrade",
		Short: "Check for upgrades of resources if present",
		Long:  ``,
		Annotations: map[string]string{
			"commandType": "main",
		},
		SilenceUsage: true,
	}
	cmd.AddCommand(
		commandForKind("task", opts),
	)

	cmd.PersistentFlags().StringVarP(&opts.kc.Path, "kubeconfig", "k", "", "Kubectl config file (default: $HOME/.kube/config)")
	cmd.PersistentFlags().StringVarP(&opts.kc.Context, "context", "c", "", "Name of the kubeconfig context to use (default: kubectl config current-context)")
	cmd.PersistentFlags().StringVarP(&opts.kc.Namespace, "namespace", "n", "", "Namespace to use (default: from $KUBECONFIG)")

	return cmd
}

// commandForKind creates a cobra.Command that when run sets
// opts.Kind and opts.Args and invokes opts.run
func commandForKind(kind string, opts *options) *cobra.Command {

	return &cobra.Command{
		Use:          kind,
		Short:        "Check updates for " + cases.Title(language.English).String(kind) + " installed via Hub CLI",
		Long:         ``,
		SilenceUsage: true,
		Example:      examples(kind),
		Annotations: map[string]string{
			"commandType": "main",
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			opts.kind = kind
			opts.args = args
			return opts.run()
		},
	}
}

func (opts *options) run() error {
	// Todo: support check-upgrade sub command for artifact type
	if opts.cli.Hub().GetType() == hub.ArtifactHubType {
		return fmt.Errorf("check-upgrade sub command is not supported for artifact type")
	}

	var err error
	if opts.cs == nil {
		opts.cs, err = kube.NewClientSet(opts.kc)
		if err != nil {
			return err
		}
	}

	resInstaller := installer.New(opts.cs)

	// List all Tekton resources installed on Cluster in a particular namespace
	resources, _ := resInstaller.ListInstalled(opts.kind, opts.cs.Namespace())

	hubClient := opts.cli.Hub()

	// Init 2 arrays to store respective data
	nonHubResources := make([]string, 0)
	hubResources := make([]hubRes, 0)

	for _, resource := range resources {

		opts.version = resource.GetLabels()[versionLabel]

		// Check whether hub label is present or not
		resourceCatalogLabel := resource.GetLabels()[hubLabel]

		// If not hub resource then add that resource to nonHubResources array
		if resourceCatalogLabel == "" {
			nonHubResources = append(nonHubResources, resource.GetName())
			continue
		}

		// Call the endpoint /resource/<catalog>/<kind>/<name>?pipelinesversion=<pipelinesversion>
		res := hubClient.GetResource(hub.ResourceOption{
			Name:            resource.GetName(),
			Catalog:         resourceCatalogLabel,
			Kind:            opts.kind,
			PipelineVersion: resInstaller.GetPipelineVersion(),
		})

		resourceDetails, err := res.Resource()
		if err != nil {
			return err
		}

		hubResource := resourceDetails.(hub.ResourceData)

		// Check if higher version available
		if opts.version < *hubResource.LatestVersion.Version {
			hubRes := hubRes{
				Name:           *hubResource.Name,
				Catalog:        cases.Title(language.English).String(resourceCatalogLabel),
				CurrentVersion: opts.version,
				LatestVersion:  *hubResource.LatestVersion.Version,
			}
			hubResources = append(hubResources, hubRes)
		}
	}

	tmplData := templateData{
		Kind:                     opts.kind,
		NonHubResources:          nonHubResources,
		HubResources:             hubResources,
		IsPipelineVersionUnknown: resInstaller.GetPipelineVersion() == "",
	}

	out := opts.cli.Stream().Out

	return printer.New(out).Tabbed(tmpl, tmplData)
}

func examples(kind string) string {
	replacer := strings.NewReplacer("%s", kind, "%S", cases.Title(language.English).String(kind))
	return replacer.Replace(cmdExamples)
}
