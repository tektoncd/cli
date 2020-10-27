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

package formatter

import (
	"fmt"
	"strings"

	"github.com/tektoncd/hub/api/gen/http/resource/client"
)

// FormatName returns name of resource with its latest version
func FormatName(name, latestVersion string) string {
	return fmt.Sprintf("%s (%s)", name, latestVersion)
}

// FormatDesc returns first 40 char of resource description
func FormatDesc(desc string) string {

	if desc == "" {
		return "---"
	} else if len(desc) > 40 {
		return desc[0:39] + "..."
	}
	return desc
}

// FormatTags returns list of tags seperated by comma
func FormatTags(tags []*client.TagResponseBody) string {
	var sb strings.Builder
	if len(tags) == 0 {
		return "---"
	}
	for i, t := range tags {
		if i != len(tags)-1 {
			sb.WriteString(strings.Trim(*t.Name, " ") + ", ")
			continue
		}
		sb.WriteString(strings.Trim(*t.Name, " "))
	}
	return sb.String()
}
