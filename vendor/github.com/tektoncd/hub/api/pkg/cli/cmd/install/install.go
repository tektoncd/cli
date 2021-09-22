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

package install

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
	defaultCatalog        = "tekton"
	versionLabel          = "app.kubernetes.io/version"
	deprecationAnnotation = "tekton.dev/deprecated"
)

type options struct {
	cli      app.CLI
	from     string
	version  string
	kind     string
	args     []string
	kc       kube.Config
	cs       kube.ClientSet
	hubRes   hub.ResourceResult
	resource *unstructured.Unstructured
}

var cmdExamples string = `
Install a %S of name 'foo':

    tkn hub install %s foo

or

Install a %S of name 'foo' of version '0.3' from Catalog 'Tekton':

    tkn hub install %s foo --version 0.3 --from tekton
`

func Command(cli app.CLI) *cobra.Command {

	opts := &options{cli: cli}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install a resource from a catalog by its kind, name and version",
		Long:  ``,
		Annotations: map[string]string{
			"commandType": "main",
		},
		SilenceUsage: true,
	}
	cmd.AddCommand(
		commandForKind("task", opts),
	)

	cmd.PersistentFlags().StringVar(&opts.from, "from", defaultCatalog, "Name of Catalog to which resource belongs.")
	cmd.PersistentFlags().StringVar(&opts.version, "version", "", "Version of Resource")

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
		Short:        "Install " + strings.Title(kind) + " from a catalog by its name and version",
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

	hubClient := opts.cli.Hub()
	opts.hubRes = hubClient.GetResource(hub.ResourceOption{
		Name:    opts.name(),
		Catalog: opts.from,
		Kind:    opts.kind,
		Version: opts.version,
	})

	manifest, err := opts.hubRes.Manifest()
	if err != nil {
		return opts.isResourceNotFoundError(err)
	}

	// This allows fake clients to be inserted while testing
	if opts.cs == nil {
		opts.cs, err = kube.NewClientSet(opts.kc)
		if err != nil {
			return err
		}
	}

	out := opts.cli.Stream().Out

	resourceInstaller := installer.New(opts.cs)

	var errors []error
	opts.resource, errors = resourceInstaller.Install(manifest, opts.from, opts.cs.Namespace())

	if len(errors) != 0 {
		resourcePipelineMinVersion, err := opts.hubRes.MinPipelinesVersion()
		if err != nil {
			return err
		}
		// process the errors and return the response
		if errors[0] == installer.ErrWarnVersionNotFound && len(errors) == 1 {
			_ = printer.New(out).String("WARN: tekton pipelines version unknown, this resource is compatible with pipelines min version v" + resourcePipelineMinVersion)
		} else {
			return opts.errors(resourceInstaller.GetPipelineVersion(), errors)
		}
	}

	if opts.resource.GetAnnotations()[deprecationAnnotation] == "true" {
		_ = printer.New(out).String("WARN: This version has been deprecated")
	}

	return printer.New(out).String(msg(opts.resource))
}

func msg(res *unstructured.Unstructured) string {
	version := res.GetLabels()[versionLabel]
	return fmt.Sprintf("%s %s(%s) installed in %s namespace",
		strings.Title(res.GetKind()), res.GetName(), version, res.GetNamespace())
}

func (opts *options) validate() error {
	return flag.ValidateVersion(opts.version)
}

func (opts *options) name() string {
	return strings.TrimSpace(opts.args[0])
}

func (opts *options) isResourceNotFoundError(err error) error {
	if err.Error() == "No Resource Found" {
		res := opts.name()
		if opts.version != "" {
			res = res + fmt.Sprintf("(%s)", opts.version)
		}
		return fmt.Errorf("%s %s from %s catalog not found in Hub", strings.Title(opts.kind), res, opts.from)
	}
	return err
}

func (opts *options) errors(pipelinesVersion string, errors []error) error {

	resourcePipelineMinVersion, vErr := opts.hubRes.MinPipelinesVersion()
	if vErr != nil {
		return vErr
	}

	resourceVersion, hubErr := opts.hubRes.ResourceVersion()
	if hubErr != nil {
		return opts.isResourceNotFoundError(hubErr)
	}

	for _, err := range errors {
		if err == installer.ErrAlreadyExist {
			existingVersion, ok := opts.resource.GetLabels()[versionLabel]
			if ok {

				switch {
				case existingVersion == resourceVersion:
					return fmt.Errorf("%s %s(%s) already exists in %s namespace. Use reinstall command to overwrite existing",
						strings.Title(opts.resource.GetKind()), opts.resource.GetName(), existingVersion, opts.cs.Namespace())

				case existingVersion < resourceVersion:
					if opts.version == "" {
						resourceVersion = resourceVersion + "(latest)"
					}
					return fmt.Errorf("%s %s(%s) already exists in %s namespace. Use upgrade command to install v%s",
						strings.Title(opts.resource.GetKind()), opts.resource.GetName(), existingVersion, opts.cs.Namespace(), resourceVersion)

				case existingVersion > resourceVersion:
					return fmt.Errorf("%s %s(%s) already exists in %s namespace. Use downgrade command to install v%s",
						strings.Title(opts.resource.GetKind()), opts.resource.GetName(), existingVersion, opts.cs.Namespace(), resourceVersion)

				default:
					return fmt.Errorf("%s %s(%s) already exists in %s namespace. Use reinstall command to overwrite existing",
						strings.Title(opts.resource.GetKind()), opts.resource.GetName(), existingVersion, opts.cs.Namespace())
				}

			} else {
				return fmt.Errorf("%s %s already exists in %s namespace but seems to be missing version label. Use reinstall command to overwrite existing",
					strings.Title(opts.resource.GetKind()), opts.resource.GetName(), opts.cs.Namespace())
			}
		}

		if err == installer.ErrVersionIncompatible {
			return fmt.Errorf("%s %s(%s) requires Tekton Pipelines min version v%s but found %s", strings.Title(opts.kind), opts.name(), resourceVersion, resourcePipelineMinVersion, pipelinesVersion)
		}

		if strings.Contains(err.Error(), "mutation failed: cannot decode incoming new object") {
			return fmt.Errorf("%v \nMake sure the pipeline version you are running is not lesser than %s and %s have correct spec fields",
				err, resourcePipelineMinVersion, opts.kind)
		}
	}

	return errors[0]
}

func examples(kind string) string {
	replacer := strings.NewReplacer("%s", kind, "%S", strings.Title(kind))
	return replacer.Replace(cmdExamples)
}
