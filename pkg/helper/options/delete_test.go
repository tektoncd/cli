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
	"os"
	"strings"
	"testing"

	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/test"
)

func TestDeleteOptions(t *testing.T) {

	testParams := []struct {
		name           string
		opt            *DeleteOptions
		stream         *cli.Stream
		resourcesNames []string
		wantError      bool
		want           string
	}{
		{
			name:           "Default Option",
			opt:            &DeleteOptions{Resource: "testRes", ForceDelete: false, DeleteAll: false},
			stream:         &cli.Stream{In: strings.NewReader("y"), Out: os.Stdout},
			resourcesNames: []string{"test"},
			wantError:      false,
			want:           "",
		},
		{
			name:           "Specify ForceDelete flag, answer yes",
			opt:            &DeleteOptions{Resource: "testRes", ForceDelete: true, DeleteAll: false},
			stream:         &cli.Stream{In: strings.NewReader("y"), Out: os.Stdout},
			resourcesNames: []string{"test"},
			wantError:      false,
			want:           "",
		},
		{
			name:           "Specify ForceDelete flag, answer no",
			opt:            &DeleteOptions{Resource: "testRes", ForceDelete: true, DeleteAll: false},
			stream:         &cli.Stream{In: strings.NewReader("n"), Out: os.Stdout},
			resourcesNames: []string{"test"},
			wantError:      false,
			want:           "",
		},
		{
			name:           "Specify DeleteAll flag, answer yes",
			opt:            &DeleteOptions{Resource: "testRes", ForceDelete: false, DeleteAll: true},
			stream:         &cli.Stream{In: strings.NewReader("y"), Out: os.Stdout},
			resourcesNames: []string{"test"},
			wantError:      false,
			want:           "",
		},
		{
			name:           "Specify DeleteAll flag, answer no",
			opt:            &DeleteOptions{Resource: "testRes", ForceDelete: false, DeleteAll: true},
			stream:         &cli.Stream{In: strings.NewReader("n"), Out: os.Stdout},
			resourcesNames: []string{"test"},
			wantError:      true,
			want:           "canceled deleting testRes \"test\"",
		},
		{
			name:           "Specify multiple resources, answer yes",
			opt:            &DeleteOptions{Resource: "testRes", ForceDelete: false, DeleteAll: false},
			stream:         &cli.Stream{In: strings.NewReader("y"), Out: os.Stdout},
			resourcesNames: []string{"test1", "test2"},
			wantError:      false,
			want:           "",
		},
		{
			name:           "Specify multiple resources, answer no",
			opt:            &DeleteOptions{Resource: "testRes", ForceDelete: false, DeleteAll: false},
			stream:         &cli.Stream{In: strings.NewReader("n"), Out: os.Stdout},
			resourcesNames: []string{"test1", "test2"},
			wantError:      true,
			want:           "canceled deleting testRes \"test1\", \"test2\"",
		},
		{
			name:           "Specify parent resource, answer y",
			opt:            &DeleteOptions{Resource: "testRes", ParentResource: "testParentRes", ParentResourceName: "my-test-resource-parent"},
			stream:         &cli.Stream{In: strings.NewReader("y"), Out: os.Stdout},
			resourcesNames: []string{""},
			wantError:      false,
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			err := tp.opt.CheckOptions(tp.stream, tp.resourcesNames)
			if tp.wantError {
				if err == nil {
					t.Fatal("error expected here")
				}
				test.AssertOutput(t, tp.want, err.Error())
			} else if err != nil {
				t.Fatalf("unexpected Error: %v", err)
			}
		})
	}
}
