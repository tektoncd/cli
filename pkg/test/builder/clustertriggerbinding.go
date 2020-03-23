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
	"time"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	tb "github.com/tektoncd/triggers/test/builder"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterTriggerBindingCreationTime sets CreationTimestamp of the ClusterTriggerBinding
func ClusterTriggerBindingCreationTime(t time.Time) tb.ClusterTriggerBindingOp {
	return func(clusterTriggerbinding *v1alpha1.ClusterTriggerBinding) {
		clusterTriggerbinding.CreationTimestamp = metav1.Time{Time: t}
	}
}
