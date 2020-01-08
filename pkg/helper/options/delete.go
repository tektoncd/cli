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
	"github.com/tektoncd/cli/pkg/helper/names"
)

type DeleteOptions struct {
	Resource    string
	ForceDelete bool
	DeleteAll   bool
}

func (o *DeleteOptions) CheckOptions(s *cli.Stream, resourceNames []string) error {
	if o.ForceDelete {
		return nil
	}

	formattedNames := names.QuotedList(resourceNames)
	if o.DeleteAll {
		fmt.Fprintf(s.Out, "Are you sure you want to delete %s and related resources %s (y/n): ", o.Resource, formattedNames)
	} else {
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
