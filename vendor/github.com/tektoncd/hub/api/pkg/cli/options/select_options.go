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

package options

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/tektoncd/hub/api/pkg/cli/app"
	"github.com/tektoncd/hub/api/pkg/cli/hub"
)

type Options struct {
	AskOpts   survey.AskOpt
	Cli       app.CLI
	Name      string
	From      string
	Kind      string
	Version   string
	HubClient hub.Client
}

func (opts *Options) Ask(resourceInfo string, options []string) error {
	var ans string
	var qs = []*survey.Question{
		{
			Name: resourceInfo,
			Prompt: &survey.Select{
				Message: fmt.Sprintf("Select %s:", resourceInfo),
				Options: options,
			},
		},
	}

	if err := survey.Ask(qs, &ans, opts.AskOpts); err != nil {
		return err
	}

	switch resourceInfo {
	case "catalog":
		opts.From = ans
	case "task":
		opts.Name = ans
	case "version":
		opts.Version = ans
	}

	return nil
}

func (opts *Options) AskCatalogName() (string, error) {
	// Get all Catalogs
	catalog, err := opts.HubClient.GetCatalogsList()
	if err != nil {
		return "", err
	}
	switch len(catalog) {
	case 0:
		return "", fmt.Errorf("No catalogs found")
	case 1:
		return catalog[0], nil
	default:
		// Ask the catalog
		err = opts.Ask("catalog", catalog)
		if err != nil {
			return "", err
		}
		return opts.From, nil
	}

}

func (opts *Options) AskResourceName() (string, error) {
	// Get all resources from the Catalog selected
	resources, err := opts.HubClient.GetResourcesList(hub.SearchOption{
		Kinds:   []string{opts.Kind},
		Catalog: opts.From,
	})
	if err != nil {
		return "", err
	}
	switch len(resources) {
	case 0:
		return "", fmt.Errorf("No resources found")
	case 1:
		return resources[0], nil
	default:
		// Ask the resource name
		err = opts.Ask(opts.Kind, resources)
		if err != nil {
			return "", err
		}
		return opts.Name, nil
	}
}

func (opts *Options) AskVersion(name string) (string, error) {
	// Get all the versions of the resource selected
	ver, err := opts.HubClient.GetResourceVersionslist(hub.ResourceOption{
		Name:    name,
		Kind:    opts.Kind,
		Catalog: opts.From,
	})
	if err != nil {
		return "", err
	}
	switch len(ver) {
	case 1:
		return ver[0], nil
	default:
		latestVersion := ver[0]
		ver[0] = ver[0] + " (latest)"

		// Ask the version
		err = opts.Ask("version", ver)
		if err != nil {
			return "", err
		}

		if strings.Contains(opts.Version, "(latest)") {
			opts.Version = latestVersion
		}
		return opts.Version, nil
	}
}
