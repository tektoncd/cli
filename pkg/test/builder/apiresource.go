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

package builder

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func APIResourceList(version, kind string) []*metav1.APIResourceList {
	group := "tekton.dev"
	return []*metav1.APIResourceList{
		{TypeMeta: metav1.TypeMeta{
			Kind:       kind,
			APIVersion: group + "/" + version,
		},
			GroupVersion: group + "/" + version,
			APIResources: []metav1.APIResource{
				{
					Name:  kind + "s",
					Group: group,
				},
			},
		},
	}
}
