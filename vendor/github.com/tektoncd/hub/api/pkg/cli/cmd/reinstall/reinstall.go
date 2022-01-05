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

package reinstall

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
Reinstall a %S of name 'foo':

    tkn hub reinstall %s foo

or

Reinstall a %S of name 'foo' of version '0.3' from Catalog 'Tekton':
	
	tkn hub reinstall %s foo --version 0.3 --from tekton
`

func Command(cli app.CLI) *cobra.Command {

	opts := &options{cli: cli}

	cmd := &cobra.Command{
		Use:   "reinstall",
		Short: "Reinstall a resource by its kind and name",
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
		Short:        "Reinstall a " + strings.Title(kind) + " by its name",
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

	installer := installer.New(opts.cs)
	opts.resource, err = installer.LookupInstalled(opts.name(), opts.kind, opts.cs.Namespace())
	if err != nil {
		if err = opts.lookupError(err); err != nil {
			return err
		}
	}

	hubClient := opts.cli.Hub()
	opts.hubRes = hubClient.GetResource(hub.ResourceOption{
		Name:    opts.name(),
		Catalog: opts.resCatalog(),
		Kind:    opts.kind,
		Version: opts.resVersion(),
	})

	manifest, err := opts.hubRes.Manifest()
	if err != nil {
		return opts.isResourceNotFoundError(err)
	}

	opts.resource, err = installer.Update(manifest, opts.from, opts.cs.Namespace())
	if err != nil {
		return opts.errors(err)
	}

	out := opts.cli.Stream().Out
	return printer.New(out).String(msg(opts.resource))
}

func msg(res *unstructured.Unstructured) string {
	version := res.GetLabels()["app.kubernetes.io/version"]
	return fmt.Sprintf("%s %s(%s) reinstalled in %s namespace",
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

func (opts *options) lookupError(err error) error {

	switch err {
	case installer.ErrNotFound:
		return fmt.Errorf("%s %s doesn't exists in %s namespace. Use install command to install the %s",
			strings.Title(opts.kind), opts.name(), opts.cs.Namespace(), opts.kind)

	case installer.ErrVersionAndCatalogMissing:
		if opts.version == "" {
			return fmt.Errorf("existing %s seems to be missing version and catalog label. Use --version & --catalog (Default: tekton) flag to reinstall the %s",
				opts.kind, opts.kind)
		}
		return nil

	case installer.ErrVersionMissing:
		if opts.version == "" {
			return fmt.Errorf("existing %s seems to be missing version label. Use --version flag to reinstall the %s",
				opts.kind, opts.kind)
		}
		return nil

	// Skip catalog missing error and use default catalog
	case installer.ErrCatalogMissing:
		return nil

	default:
		return err
	}
}

func (opts *options) errors(err error) error {

	if err == installer.ErrNotFound {
		return fmt.Errorf("%s %s doesn't exists in %s namespace. Use install command to install the resource",
			strings.Title(opts.kind), opts.name(), opts.cs.Namespace())
	}

	if strings.Contains(err.Error(), "mutation failed: cannot decode incoming new object") {
		version, vErr := opts.hubRes.MinPipelinesVersion()
		if vErr != nil {
			return vErr
		}
		return fmt.Errorf("%v \nMake sure the pipeline version you are running is not lesser than %s and %s have correct spec fields",
			err, version, opts.kind)
	}
	return err
}

func (opts *options) resCatalog() string {
	labels := opts.resource.GetLabels()
	if len(labels) == 0 {
		return opts.from
	}
	catalog, ok := labels[catalogLabel]
	if ok {
		if catalog != opts.from && opts.from != "" {
			return opts.from
		}
		return catalog
	}
	return opts.from
}

func (opts *options) resVersion() string {
	labels := opts.resource.GetLabels()
	if len(labels) == 0 {
		return opts.version
	}
	version, ok := labels[versionLabel]
	if ok {
		if version != opts.version && opts.version != "" {
			return opts.version
		}
		return version
	}
	return opts.version
}

func examples(kind string) string {
	replacer := strings.NewReplacer("%s", kind, "%S", strings.Title(kind))
	return replacer.Replace(cmdExamples)
}
