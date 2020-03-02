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

package options

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/names"
)

type DeleteOptions struct {
	Resource           string
	ParentResource     string
	ParentResourceName string
	ForceDelete        bool
	DeleteRelated      bool
	DeleteAllNs        bool
	DeleteAll          bool
	Keep               int
}

func (o *DeleteOptions) CheckOptions(s *cli.Stream, resourceNames []string, ns string) error {
	if o.Keep > 0 && !(o.DeleteAllNs || o.DeleteAll) {
		return fmt.Errorf("must use --all flag with --keep")
	}
	// make sure no resource names are provided when using --all flag
	if len(resourceNames) > 0 && (o.DeleteAllNs || o.DeleteAll) {
		return fmt.Errorf("--all flag should not have any arguments or flags specified with it")
	}

	// make sure either resource names are provided, name of related resource,
	// or --all specified if deleting PipelineRuns or TaskRuns
	if len(resourceNames) == 0 && o.ParentResource != "" && o.ParentResourceName == "" && !o.DeleteAllNs {
		return fmt.Errorf("must provide %s name(s) or use --%s flag or --all flag to use delete", o.Resource, o.ParentResource)
	}

	// make sure that resource name or --all flag is specified to use delete
	// in non PipelineRun or TaskRun deletions
	if len(resourceNames) == 0 && o.ParentResource == "" && o.ParentResourceName == "" && !o.DeleteAllNs && !o.DeleteAll {
		return fmt.Errorf("must provide %s name(s) or use --all flag with delete", o.Resource)
	}

	if o.Keep < 0 {
		return fmt.Errorf("keep option should not be lower than 0")
	}

	if o.ForceDelete {
		return nil
	}

	formattedNames := names.QuotedList(resourceNames)

	keepStr := ""
	if o.Keep > 0 {
		keepStr = fmt.Sprintf(" keeping %d %ss", o.Keep, o.Resource)
	}
	switch {
	case o.DeleteAllNs:
		fmt.Fprintf(s.Out, "Are you sure you want to delete all %ss in namespace %q%s (y/n): ", o.Resource, ns, keepStr)
	case o.DeleteAll:
		fmt.Fprintf(s.Out, "Are you sure you want to delete all %ss%s (y/n): ", o.Resource, keepStr)
	case o.ParentResource != "" && o.ParentResourceName != "":
		fmt.Fprintf(s.Out, "Are you sure you want to delete all %ss related to %s %q (y/n): ", o.Resource, o.ParentResource, o.ParentResourceName)
	case o.DeleteRelated:
		fmt.Fprintf(s.Out, "Are you sure you want to delete %s and related resources %s (y/n): ", o.Resource, formattedNames)
	default:
		fmt.Fprintf(s.Out, "Are you sure you want to delete %s %s (y/n): ", o.Resource, formattedNames)
	}

	scanner := bufio.NewScanner(s.In)
	for scanner.Scan() {
		t := strings.TrimSpace(scanner.Text())
		if t == "y" {
			break
		} else if t == "n" {
			return fmt.Errorf("canceled deleting %s %s", o.Resource, formattedNames)
		}
		fmt.Fprint(s.Out, "Please enter (y/n): ")
	}

	return nil
}
