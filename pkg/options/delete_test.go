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
			opt:            &DeleteOptions{Resource: "testRes", ForceDelete: false, DeleteRelated: false, DeleteAllNs: false},
			stream:         &cli.Stream{In: strings.NewReader("y"), Out: os.Stdout},
			resourcesNames: []string{"test"},
			wantError:      false,
			want:           "",
		},
		{
			name:           "Specify ForceDelete flag, answer yes",
			opt:            &DeleteOptions{Resource: "testRes", ForceDelete: true, DeleteRelated: false},
			stream:         &cli.Stream{In: strings.NewReader("y"), Out: os.Stdout},
			resourcesNames: []string{"test"},
			wantError:      false,
			want:           "",
		},
		{
			name:           "Specify ForceDelete flag, answer no",
			opt:            &DeleteOptions{Resource: "testRes", ForceDelete: true, DeleteRelated: false},
			stream:         &cli.Stream{In: strings.NewReader("n"), Out: os.Stdout},
			resourcesNames: []string{"test"},
			wantError:      false,
			want:           "",
		},
		{
			name:           "Specify DeleteRelated flag, answer yes",
			opt:            &DeleteOptions{Resource: "testRes", ForceDelete: false, DeleteRelated: true},
			stream:         &cli.Stream{In: strings.NewReader("y"), Out: os.Stdout},
			resourcesNames: []string{"test"},
			wantError:      false,
			want:           "",
		},
		{
			name:           "Specify DeleteRelated flag, answer no",
			opt:            &DeleteOptions{Resource: "testRes", ForceDelete: false, DeleteRelated: true},
			stream:         &cli.Stream{In: strings.NewReader("n"), Out: os.Stdout},
			resourcesNames: []string{"test"},
			wantError:      true,
			want:           "canceled deleting testRes(s) \"test\"",
		},
		{
			name:           "Specify multiple resources, answer yes",
			opt:            &DeleteOptions{Resource: "testRes", ForceDelete: false, DeleteRelated: false},
			stream:         &cli.Stream{In: strings.NewReader("y"), Out: os.Stdout},
			resourcesNames: []string{"test1", "test2"},
			wantError:      false,
			want:           "",
		},
		{
			name:           "Specify multiple resources, answer no",
			opt:            &DeleteOptions{Resource: "testRes", ForceDelete: false, DeleteRelated: false},
			stream:         &cli.Stream{In: strings.NewReader("n"), Out: os.Stdout},
			resourcesNames: []string{"test1", "test2"},
			wantError:      true,
			want:           "canceled deleting testRes(s) \"test1\", \"test2\"",
		},
		{
			name:           "Specify parent resource, answer y",
			opt:            &DeleteOptions{Resource: "testRes", ParentResource: "testParentRes", ParentResourceName: "my-test-resource-parent"},
			stream:         &cli.Stream{In: strings.NewReader("y"), Out: os.Stdout},
			resourcesNames: []string{""},
			wantError:      false,
		},
		{
			name:           "Specify DeleteAllNs option",
			opt:            &DeleteOptions{DeleteAllNs: true},
			stream:         &cli.Stream{In: strings.NewReader("y"), Out: os.Stdout},
			resourcesNames: []string{},
			wantError:      false,
		},
		{
			name:           "Error when all defaults specified with ParentResource",
			opt:            &DeleteOptions{Resource: "TaskRun", ParentResource: "task", ForceDelete: false, DeleteRelated: false, DeleteAllNs: false},
			resourcesNames: []string{},
			wantError:      true,
			want:           "must provide TaskRun name(s) or use --task flag or --all flag to use delete",
		},
		{
			name:           "Error when resource name provided with DeleteAllNs",
			opt:            &DeleteOptions{DeleteAllNs: true},
			resourcesNames: []string{"test1"},
			wantError:      true,
			want:           "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:           "Error when DeleteRelated with DeleteAllNs",
			opt:            &DeleteOptions{DeleteAllNs: true, DeleteRelated: true},
			resourcesNames: []string{"test1"},
			wantError:      true,
			want:           "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:           "Error when resource name used with keep",
			opt:            &DeleteOptions{Keep: 1},
			resourcesNames: []string{"test1"},
			wantError:      true,
			want:           "--keep flag should not have any arguments specified with it",
		},
		{
			name:           "Specify DeleteAll option",
			opt:            &DeleteOptions{DeleteAll: true},
			stream:         &cli.Stream{In: strings.NewReader("y"), Out: os.Stdout},
			resourcesNames: []string{},
			wantError:      false,
		},
		{
			name:           "Error when resource name provided with DeleteAll",
			opt:            &DeleteOptions{DeleteAll: true},
			resourcesNames: []string{"test1"},
			wantError:      true,
			want:           "--all flag should not have any arguments or flags specified with it",
		},
		{
			name:           "Error when not specifying resource name or --all flag in non PipelineRun/TaskRun deletion",
			opt:            &DeleteOptions{Resource: "Condition", ParentResource: "", ParentResourceName: "", DeleteRelated: false, DeleteAllNs: false},
			resourcesNames: []string{},
			wantError:      true,
			want:           "must provide Condition name(s) or use --all flag with delete",
		},
		{
			name:           "Specify KeepSince, ParentResource, ParentResourceName and Resource",
			opt:            &DeleteOptions{KeepSince: 20, ParentResource: "Task", ParentResourceName: "foobar", Resource: "TaskRun"},
			resourcesNames: []string{},
			wantError:      false,
			stream:         &cli.Stream{In: strings.NewReader("y"), Out: os.Stdout},
			want:           "Are you sure you want to delete all TaskRuns related to Task \"foobar\" except for ones created in last 20 minutes (y/n)",
		},
		{
			name:           "Specify Keep, ParentResource, ParentResourceName and Resource",
			opt:            &DeleteOptions{Keep: 5, ParentResource: "Task", ParentResourceName: "foobar", Resource: "TaskRun"},
			resourcesNames: []string{},
			wantError:      false,
			stream:         &cli.Stream{In: strings.NewReader("y"), Out: os.Stdout},
			want:           "Are you sure you want to delete all TaskRuns related to Task \"foobar\" keeping 5 TaskRuns (y/n)",
		},
		{
			name:           "Specify KeepSince, Keep, ParentResource, ParentResourceName and Resource",
			opt:            &DeleteOptions{KeepSince: 20, Keep: 5, ParentResource: "Task", ParentResourceName: "foobar", Resource: "TaskRun"},
			resourcesNames: []string{},
			wantError:      false,
			stream:         &cli.Stream{In: strings.NewReader("y"), Out: os.Stdout},
			want:           "Are you sure you want to delete all TaskRuns related to Task \"foobar\" except for ones created in last 20 minutes and keeping 5 TaskRuns (y/n)",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			err := tp.opt.CheckOptions(tp.stream, tp.resourcesNames, "")
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
