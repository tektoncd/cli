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

package upgrade

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tektoncd/hub/api/pkg/cli/app"
	"github.com/tektoncd/hub/api/pkg/cli/flag"
	"github.com/tektoncd/hub/api/pkg/cli/hub"
	"github.com/tektoncd/hub/api/pkg/cli/installer"
	"github.com/tektoncd/hub/api/pkg/cli/kube"
	"github.com/tektoncd/hub/api/pkg/cli/printer"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	defaultCatalog = "tekton"
	catalogLabel   = "hub.tekton.dev/catalog"
	versionLabel   = "app.kubernetes.io/version"
)

type options struct {
	cli      app.CLI
	version  string
	kind     string
	args     []string
	kc       kube.Config
	cs       kube.ClientSet
	hubRes   hub.ResourceResult
	resource *unstructured.Unstructured
}

var cmdExamples string = `
Upgrade a %S of name 'foo':

    tkn hub upgrade %s foo

or

Upgrade a %S of name 'foo' to version '0.3':

    tkn hub upgrade %s foo --to 0.3
`

func Command(cli app.CLI) *cobra.Command {

	opts := &options{cli: cli}

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade an installed resource",
		Long:  ``,
		Annotations: map[string]string{
			"commandType": "main",
		},
		SilenceUsage: true,
	}
	cmd.AddCommand(
		commandForKind("task", opts),
	)

	cmd.PersistentFlags().StringVar(&opts.version, "to", "", "Version of Resource")

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
		Short:        "Upgrade a " + strings.Title(kind) + " by its name",
		Long:         ``,
		SilenceUsage: true,
		Example:      examples(kind),
		Annotations: map[string]string{
			"commandType": "main",
		},
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.kind = kind
			opts.args = args
			return opts.run()
		},
	}
}

func (opts *options) run() error {

	if err := opts.validate(); err != nil {
		return err
	}

	// This allows fake clients to be inserted while testing
	var err error
	if opts.cs == nil {
		opts.cs, err = kube.NewClientSet(opts.kc)
		if err != nil {
			return err
		}
	}

	resInstaller := installer.New(opts.cs)
	opts.resource, err = resInstaller.LookupInstalled(opts.name(), opts.kind, opts.cs.Namespace())
	if err != nil {
		if err = opts.lookupError(err); err != nil {
			return err
		}
	}

	catalog := opts.resCatalog()

	hubClient := opts.cli.Hub()
	opts.hubRes = hubClient.GetResource(hub.ResourceOption{
		Name:    opts.name(),
		Catalog: catalog,
		Kind:    opts.kind,
		Version: opts.version,
	})

	manifest, err := opts.hubRes.Manifest()
	if err != nil {
		return err
	}

	out := opts.cli.Stream().Out

	var errors []error
	opts.resource, errors = resInstaller.Upgrade(manifest, catalog, opts.cs.Namespace())
	if len(errors) != 0 {

		resourcePipelineMinVersion, vErr := opts.hubRes.MinPipelinesVersion()
		if vErr != nil {
			return vErr
		}

		if errors[0] == installer.ErrWarnVersionNotFound && len(errors) == 1 {
			_ = printer.New(out).String("WARN: tekton pipelines version unknown, this resource is compatible with pipelines min version v" + resourcePipelineMinVersion)
		} else {
			return opts.errors(resourcePipelineMinVersion, resInstaller.GetPipelineVersion(), errors)
		}
	}

	return printer.New(out).String(msg(opts.resource))
}

func msg(res *unstructured.Unstructured) string {
	version := res.GetLabels()[versionLabel]
	return fmt.Sprintf("%s %s upgraded to v%s in %s namespace",
		strings.Title(res.GetKind()), res.GetName(), version, res.GetNamespace())
}

func (opts *options) validate() error {
	return flag.ValidateVersion(opts.version)
}

func (opts *options) name() string {
	return strings.TrimSpace(opts.args[0])
}

func examples(kind string) string {
	replacer := strings.NewReplacer("%s", kind, "%S", strings.Title(kind))
	return replacer.Replace(cmdExamples)
}

func (opts *options) resCatalog() string {
	labels := opts.resource.GetLabels()
	if len(labels) == 0 {
		return defaultCatalog
	}
	catalog, ok := labels[catalogLabel]
	if ok {
		return catalog
	}
	return defaultCatalog
}

func (opts *options) lookupError(err error) error {

	switch err {
	case installer.ErrNotFound:
		return fmt.Errorf("%s %s doesn't exist in %s namespace. Use install command to install the %s",
			strings.Title(opts.kind), opts.name(), opts.cs.Namespace(), opts.kind)

	case installer.ErrVersionAndCatalogMissing:
		return fmt.Errorf("%s %s seems to be missing version and catalog label. Use reinstall command to overwrite existing %s",
			strings.Title(opts.resource.GetKind()), opts.resource.GetName(), opts.kind)

	case installer.ErrVersionMissing:
		return fmt.Errorf("%s %s seems to be missing version label. Use reinstall command to overwrite existing %s",
			strings.Title(opts.resource.GetKind()), opts.resource.GetName(), opts.kind)

	// Skip catalog missing error and use default catalog
	case installer.ErrCatalogMissing:
		return nil

	default:
		return err
	}
}

func (opts *options) errors(resourcePipelineMinVersion, pipelinesVersion string, errors []error) error {

	newVersion, hubErr := opts.hubRes.ResourceVersion()
	if hubErr != nil {
		return hubErr
	}

	for _, err := range errors {
		if err == installer.ErrNotFound {
			return fmt.Errorf("%s %s doesn't exists in %s namespace. Use install command to install the %s",
				strings.Title(opts.kind), opts.name(), opts.cs.Namespace(), opts.kind)
		}

		if err == installer.ErrSameVersion {
			return fmt.Errorf("cannot upgrade %s %s to v%s. existing resource seems to be of same version. Use reinstall command to overwrite existing %s",
				strings.ToLower(opts.resource.GetKind()), opts.resource.GetName(), newVersion, opts.kind)
		}

		if err == installer.ErrLowerVersion {
			existingVersion := opts.resource.GetLabels()[versionLabel]
			return fmt.Errorf("cannot upgrade %s %s to v%s. existing resource seems to be of higher version(v%s). Use downgrade command",
				strings.ToLower(opts.resource.GetKind()), opts.resource.GetName(), newVersion, existingVersion)
		}

		if err == installer.ErrVersionIncompatible {
			return fmt.Errorf("cannot upgrade %s %s(%s) as it requires Tekton Pipelines min version v%s but found %s", strings.Title(opts.kind), opts.name(), newVersion, resourcePipelineMinVersion, pipelinesVersion)
		}

		if strings.Contains(err.Error(), "mutation failed: cannot decode incoming new object") {
			return fmt.Errorf("%v \nMake sure the pipeline version you are running is not lesser than %s and %s have correct spec fields",
				err, resourcePipelineMinVersion, opts.kind)
		}
	}

	return errors[0]
}
