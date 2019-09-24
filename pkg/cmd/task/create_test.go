package task

import (
	"io"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	pipelinetest "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
)

func TestTaskCreate(t *testing.T) {
	clock := clockwork.NewFakeClock()

	seeds := make([]pipelinetest.Clients, 0)
	for i := 0; i < 3; i++ {
		cs, _ := test.SeedTestData(t, pipelinetest.Data{})
		seeds = append(seeds, cs)
	}

	tasks := []*v1alpha1.Task{
		tb.Task("build-docker-image-from-git-source", "ns", cb.TaskCreationTime(clock.Now().Add(-1*time.Minute))),
	}
	cs, _ := test.SeedTestData(t, pipelinetest.Data{Tasks: tasks})
	seeds = append(seeds, cs)

	testParams := []struct {
		name        string
		command     []string
		input       pipelinetest.Clients
		inputStream io.Reader
		wantError   bool
		want        string
	}{
		{
			name:        "Create successfully",
			command:     []string{"create", "--from", "./testdata/task.yaml", "-n", "ns"},
			input:       seeds[0],
			inputStream: nil,
			wantError:   false,
			want:        "Task created: task.yaml\n",
		},
		{
			name:        "Filename does not exist",
			command:     []string{"create", "-f", "./testdata/notexist.yaml", "-n", "ns"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   true,
			want:        "open ./testdata/notexist.yaml: no such file or directory",
		},
		{
			name:        "Unsupported file type",
			command:     []string{"create", "-f", "./testdata/task.txt", "-n", "ns"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   true,
			want:        "the path ./testdata/task.txt does not match target pattern",
		},
		{
			name:        "Mismatched resource file",
			command:     []string{"create", "-f", "./testdata/taskrun.yaml", "-n", "ns"},
			input:       seeds[1],
			inputStream: nil,
			wantError:   true,
			want:        "provided TaskRun instead of Task kind",
		},
		{
			name:        "Existing task",
			command:     []string{"create", "-f", "./testdata/task.yaml", "-n", "ns"},
			input:       seeds[3],
			inputStream: nil,
			wantError:   true,
			want:        "task \"build-docker-image-from-git-source\" has already exists in namespace ns",
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := &test.Params{Tekton: tp.input.Pipeline}
			task := Command(p)

			if tp.inputStream != nil {
				task.SetIn(tp.inputStream)
			}

			out, err := test.ExecuteCommand(task, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("Error expected here")
				} else {
					test.AssertOutput(t, tp.want, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected Error")
				}
				test.AssertOutput(t, tp.want, out)
			}
		})
	}
}
