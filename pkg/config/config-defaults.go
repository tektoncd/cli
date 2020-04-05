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

package defaults

import (
	v1 "k8s.io/api/core/v1"

	"github.com/tektoncd/cli/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const pipelineNamespace = "tekton-pipelines"
const configDefaultsName = "config-defaults"

// GetPipelineVersion Get pipeline version, functions imported from Dashboard
func getConfigDefaults(c *cli.Clients) (*v1.ConfigMap, error) {
	configDefaults, err := c.Kube.CoreV1().ConfigMaps(pipelineNamespace).Get(configDefaultsName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return configDefaults, nil
}

func GetDefaultTimeout(c *cli.Clients) (string, error) {
	defaultTimeoutKey := "default-timeout-minutes"
	onehour := "1h"
	configDefaults, err := getConfigDefaults(c)
	if err != nil {
		return onehour, err
	}

	defaultTimeout := configDefaults.Data[defaultTimeoutKey]
	return defaultTimeout, nil
}
