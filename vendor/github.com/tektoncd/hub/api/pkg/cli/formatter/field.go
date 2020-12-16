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
	"github.com/tektoncd/hub/api/pkg/cli/hub"
	"golang.org/x/crypto/ssh/terminal"
)

var icons = map[string]string{
	"bullet":             "∙ ",
	"name":               "📦 ",
	"displayName":        "🗂 ",
	"version":            "📌 ",
	"description":        "📖 ",
	"minPipelineVersion": "🗒  ",
	"rating":             "⭐ ️",
	"tags":               "🏷 ",
	"install":            "⚒ ",
}

// FormatName returns name of resource with its latest version
func FormatName(name, latestVersion string) string {
	return fmt.Sprintf("%s (%s)", name, latestVersion)
}

// FormatDesc returns first 40 char of resource description
func FormatDesc(desc string, num int) string {

	if desc == "" {
		return "---"
	} else if len(desc) > num {
		return desc[0:num-1] + "..."
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

// WrapText returns description broken down in multiple lines with
// max width passed to it
// titleLength would be the length of title on left hand side before the
// text starts, so the length of text in first line would be maxwidth - titleLength
func WrapText(desc string, maxWidth, titleLength int) string {
	if desc == "" {
		return "---"
	}
	desc = strings.ReplaceAll(desc, "\n", " ")

	width, _, err := terminal.GetSize(0)
	if err != nil {
		return breakString(desc, maxWidth, titleLength)
	}

	if maxWidth <= width {
		return breakString(desc, maxWidth, titleLength)
	}
	return breakString(desc, width, titleLength)
}

func breakString(desc string, width, titleLength int) string {
	if len(desc) <= width {
		return desc
	}
	var sb strings.Builder
	descLength := len(desc)

	// First line will have title due to which it will default width - 16 char
	firstLineEnd := findSpaceIndexFromLast(desc[0 : width-titleLength])
	sb.WriteString(desc[0:firstLineEnd] + "\n")

	var spaceIndex int
	for i := firstLineEnd; i < descLength; i = i + spaceIndex {
		if descLength < i+width {
			sb.WriteString(desc[i:])
		} else {
			spaceIndex = findSpaceIndexFromLast(desc[i : i+width])
			sb.WriteString(desc[i:i+spaceIndex] + "\n")
		}
	}
	return sb.String()
}

func findSpaceIndexFromLast(str string) int {
	return strings.LastIndex(str, " ")
}

// FormatVersion returns version appended with (latest) if the
// latest field passed is true
func FormatVersion(version string, latest bool) string {
	if latest {
		return version + " (Latest)"
	}
	return version
}

// Icon returns icon for a title passed
func Icon(title string) string {
	ic, ok := icons[title]
	if ok {
		return ic
	}
	return ""
}

// DefaultValue returns default value if string is empty
func DefaultValue(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

func FormatInstallCMD(res hub.ResourceData, resVer hub.ResourceWithVersionData, latest bool) string {
	var sb strings.Builder
	sb.WriteString("tkn hub install")
	// append kind
	sb.WriteString(" " + strings.ToLower(*res.Kind))
	// append name
	sb.WriteString(" " + strings.ToLower(*res.Name))
	// append version if not latest
	if !latest {
		sb.WriteString(" --version " + *resVer.Version)
	}
	//append catalog name if not default catalog
	if *res.Catalog.Name != "tekton" {
		sb.WriteString(" --from " + *res.Catalog.Name)
	}
	return sb.String()
}
