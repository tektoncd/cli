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

package version

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/flags"
	"github.com/tektoncd/cli/pkg/version"
)

const (
	devVersion       = "dev"
	latestReleaseURL = "https://api.github.com/repos/tektoncd/cli/releases/latest"
)

var (
	// NOTE: use go build -ldflags "-X github.com/tektoncd/cli/pkg/cmd/version.clientVersion=$(git describe)"
	clientVersion = devVersion
	// flag to skip check flag in version cmd
	skipCheckFlag = "false"
	// NOTE: use go build -ldflags "-X github.com/tektoncd/cli/pkg/cmd/version.namespace=tekton-pipelines"
	namespace string

	component = ""
)

// Command returns version command
func Command(p cli.Params) *cobra.Command {
	var check bool

	var cmd = &cobra.Command{
		Use:   "version",
		Short: "Prints version information",
		Annotations: map[string]string{
			"commandType": "utility",
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := flags.InitParams(p, cmd); err != nil {
				// this check allows tkn version to be run without
				// a kubeconfig so users can verify the tkn version
				noConfigErr := strings.Contains(err.Error(), "no configuration has been provided")
				if noConfigErr {
					return nil
				}
				return err
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			cs, err := p.Clients()
			if err == nil {
				switch component {
				case "":
					fmt.Fprintf(cmd.OutOrStdout(), "Client version: %s\n", clientVersion)
					chainsVersion, _ := version.GetChainsVersion(cs, namespace)
					if chainsVersion != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "Chains version: %s\n", chainsVersion)
					}
					pipelineVersion, _ := version.GetPipelineVersion(cs, namespace)
					if pipelineVersion == "" {
						pipelineVersion = "unknown, " +
							"pipeline controller may be installed in another namespace please use tkn version -n {namespace}"
					}

					fmt.Fprintf(cmd.OutOrStdout(), "Pipeline version: %s\n", pipelineVersion)
					triggersVersion, _ := version.GetTriggerVersion(cs, namespace)
					if triggersVersion != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "Triggers version: %s\n", triggersVersion)
					}
					dashboardVersion, _ := version.GetDashboardVersion(cs, namespace)
					if dashboardVersion != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "Dashboard version: %s\n", dashboardVersion)
					}
					operatorVersion, _ := version.GetOperatorVersion(cs, namespace)
					if operatorVersion != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "Operator version: %s\n", operatorVersion)
					}
					hubVersion, _ := version.GetHubVersion(cs, namespace)
					if hubVersion != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "Hub version: %s\n", hubVersion)
					}
				case "client":
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", clientVersion)
				case "chains":
					chainsVersion, _ := version.GetChainsVersion(cs, namespace)
					if chainsVersion == "" {
						chainsVersion = "unknown"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", chainsVersion)
				case "pipeline":
					pipelineVersion, _ := version.GetPipelineVersion(cs, namespace)
					if pipelineVersion == "" {
						pipelineVersion = "unknown"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", pipelineVersion)
				case "triggers":
					triggersVersion, _ := version.GetTriggerVersion(cs, namespace)
					if triggersVersion == "" {
						triggersVersion = "unknown"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", triggersVersion)
				case "dashboard":
					dashboardVersion, _ := version.GetDashboardVersion(cs, namespace)
					if dashboardVersion == "" {
						dashboardVersion = "unknown"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", dashboardVersion)
				case "operator":
					operatorVersion, _ := version.GetOperatorVersion(cs, namespace)
					if operatorVersion == "" {
						operatorVersion = "unknown"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", operatorVersion)
				case "hub":
					hubVersion, _ := version.GetHubVersion(cs, namespace)
					if hubVersion == "" {
						hubVersion = "unknown"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", hubVersion)
				default:
					fmt.Fprintf(cmd.OutOrStdout(), "Invalid component value\n")
				}
			} else {
				switch component {
				case "":
					fmt.Fprintf(cmd.OutOrStdout(), "Client version: %s\n", clientVersion)
				case "client":
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", clientVersion)
				case "chains", "pipeline", "triggers", "dashboard", "operator":
					fmt.Fprintf(cmd.OutOrStdout(), "unknown\n")
				default:
					fmt.Fprintf(cmd.OutOrStdout(), "Invalid component value\n")
				}
			}

			if !check || clientVersion == devVersion {
				return nil
			}

			client := NewClient(time.Duration(3 * time.Second))
			output, err := checkRelease(client)
			fmt.Fprint(cmd.OutOrStdout(), output)
			return err
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", namespace,
		"namespace to check installed controller version")
	flags.AddTektonOptions(cmd)

	cmd.Flags().StringVarP(&component, "component", "", "", "provide a particular component name for its version (client|chains|pipeline|triggers|dashboard)")

	if skipCheckFlag != "true" {
		cmd.Flags().BoolVar(&check, "check", false, "check if a newer version is available")
	}

	return cmd
}

type GHVersion struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

type Option func(*Client)

type Client struct {
	httpClient *http.Client
}

func SetHTTPClient(httpClient *http.Client) Option {
	return func(cli *Client) {
		cli.httpClient = httpClient
	}
}

func NewClient(timeout time.Duration, options ...Option) *Client {
	cli := Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}

	for i := range options {
		options[i](&cli)
	}
	return &cli
}

func (cli *Client) getRelease(url string) (ghversion GHVersion, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ghversion, errors.Wrap(err, "failed to fetch the latest version")
	}

	res, err := cli.httpClient.Do(req)
	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()
	if err != nil {
		return ghversion, errors.Wrap(err, "request failed")
	}

	if res.StatusCode != http.StatusOK {
		return ghversion, fmt.Errorf("invalid http status %d, error: %s", res.StatusCode, res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ghversion, errors.Wrap(err, "failed to read the latest version response body")
	}
	response := GHVersion{}

	if err := json.Unmarshal(body, &response); err != nil {
		return ghversion, errors.Wrap(err, "failed to unmarshal the latest version response body")
	}

	return response, nil
}

func checkRelease(client *Client) (string, error) {
	response, err := client.getRelease(latestReleaseURL)
	if err != nil {
		return "", err
	}

	latest, err := parseVersion(response.TagName)
	if err != nil {
		return "", err
	}

	current, err := parseVersion(clientVersion)
	if err != nil {
		return "", err
	}

	if current.LT(*latest) {
		return fmt.Sprintf("A newer version (v%s) of Tekton CLI is available, please check %s\n", latest, response.HTMLURL), nil
	}

	return fmt.Sprintf("You are running the latest version (v%s) of Tekton CLI\n", latest), nil
}

func parseVersion(version string) (*semver.Version, error) {
	version = strings.TrimSpace(version)
	// Strip the leading 'v' in the version strings
	v, err := semver.Parse(strings.TrimLeft(version, "v"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse version")
	}
	return &v, nil
}
