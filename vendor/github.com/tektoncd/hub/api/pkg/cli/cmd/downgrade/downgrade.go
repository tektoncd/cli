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

package downgrade

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-version"
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
	cli            app.CLI
	version        string
	kind           string
	args           []string
	kc             kube.Config
	cs             kube.ClientSet
	hubRes         hub.ResourceVersionResult
	hubResVersions *hub.ResVersions
	resource       *unstructured.Unstructured
}

var cmdExamples string = `
Downgrade a %S of name 'foo' to previous version:

    tkn hub downgrade %s foo

or

Downgrade a %S of name 'foo' to version '0.3':

    tkn hub downgrade %s foo --to 0.3
`

func Command(cli app.CLI) *cobra.Command {

	opts := &options{cli: cli}

	cmd := &cobra.Command{
		Use:   "downgrade",
		Short: "Downgrade an installed resource",
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
		Short:        "Downgrade an installed " + strings.Title(kind) + " by its name to a lower version",
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

	catalog := opts.resCatalog()
	existingVersion := opts.resVersion()

	hubClient := opts.cli.Hub()
	opts.hubRes = hubClient.GetResourceVersions(hub.ResourceOption{
		Name:    opts.name(),
		Catalog: catalog,
		Kind:    opts.kind,
		Version: existingVersion,
	})

	opts.hubResVersions, err = opts.hubRes.ResourceVersions()
	if err != nil {
		return err
	}

	opts.version, err = opts.findLowerVersion(existingVersion)
	if err != nil {
		return err
	}

	manifest, err := opts.hubRes.VersionManifest(opts.version)
	if err != nil {
		return err
	}

	opts.resource, err = installer.Downgrade(manifest, catalog, opts.cs.Namespace())
	if err != nil {
		return opts.errors(err)
	}

	out := opts.cli.Stream().Out
	return printer.New(out).String(msg(opts.resource))
}

func msg(res *unstructured.Unstructured) string {
	version := res.GetLabels()[versionLabel]
	return fmt.Sprintf("%s %s downgraded to v%s in %s namespace",
		strings.Title(res.GetKind()), res.GetName(), version, res.GetNamespace())
}

func (opts *options) findLowerVersion(current string) (string, error) {
	if opts.version != "" {
		currentVer, _ := version.NewVersion(current)
		newVer, _ := version.NewVersion(opts.version)
		if currentVer.LessThan(newVer) {
			return "", fmt.Errorf("cannot downgrade %s %s to v%s. existing resource seems to be of lower version(v%s). Use upgrade command",
				opts.kind, opts.name(), opts.version, current)
		}
		return opts.version, nil
	}

	for i, v := range opts.hubResVersions.Versions {
		if current == *v.Version {
			if i == 0 {
				return "", fmt.Errorf("cannot downgrade %s %s, it seems to be at its lowest version(v%s)",
					opts.kind, opts.name(), current)
			}
			return *opts.hubResVersions.Versions[i-1].Version, nil
		}
	}
	return "", fmt.Errorf("resource version not found")
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

func (opts *options) resVersion() string {
	version, _ := opts.resource.GetLabels()[versionLabel]
	return version
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

func (opts *options) errors(err error) error {

	if err == installer.ErrNotFound {
		return fmt.Errorf("%s %s doesn't exists in %s namespace. Use install command to install the resource",
			strings.Title(opts.kind), opts.name(), opts.cs.Namespace())
	}

	if err == installer.ErrSameVersion {
		return fmt.Errorf("cannot downgrade %s %s to v%s. existing resource seems to be of same version. Use reinstall command to overwrite existing %s",
			strings.ToLower(opts.resource.GetKind()), opts.resource.GetName(), opts.version, opts.kind)
	}

	if err == installer.ErrHigherVersion {
		existingVersion, _ := opts.resource.GetLabels()[versionLabel]
		return fmt.Errorf("cannot downgrade %s %s to v%s. existing resource seems to be of lower version(v%s). Use upgrade command",
			strings.ToLower(opts.resource.GetKind()), opts.resource.GetName(), opts.version, existingVersion)
	}

	return err
}
