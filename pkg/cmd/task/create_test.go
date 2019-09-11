package task

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
			want:        "does not support such extension for ./testdata/task.txt",
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

func TestLoadRemoteFile(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		okResponse := `
apiVersion: tekton.dev/v1alpha1
kind: Task
metadata:
  name: build-docker-image-from-git-source
spec:
  inputs:
    resources:
      - name: docker-source
        type: git
    params:
      - name: pathToDockerFile
        type: string
        description: The path to the dockerfile to build
        default: /workspace/docker-source/Dockerfile
      - name: pathToContext
        type: string
        description:
          The build context used by Kaniko
          (https://github.com/GoogleContainerTools/kaniko#kaniko-build-contexts)
        default: /workspace/docker-source
  outputs:
    resources:
      - name: builtImage
        type: image
  steps:
    - name: build-and-push
      image: gcr.io/kaniko-project/executor:v0.9.0
      # specifying DOCKER_CONFIG is required to allow kaniko to detect docker credential
      env:
        - name: "DOCKER_CONFIG"
          value: "/builder/home/.docker/"
      command:
        - /kaniko/executor
      args:
        - --dockerfile=$(inputs.params.pathToDockerFile)
        - --destination=$(outputs.resources.builtImage.url)
        - --context=$(inputs.params.pathToContext)`
		_, _ = w.Write([]byte(okResponse))
	})
	httpClient, teardown := testingHTTPClient(h)
	defer teardown()

	cli := NewClient(time.Duration(0))
	cli.httpClient = httpClient
	remoteContent, _ := loadFileContent("https://foo.com/task.yaml", cli)

	localContent, _ := loadFileContent("./testdata/task.yaml", nil)

	if d := cmp.Diff(remoteContent, localContent); d != "" {
		t.Errorf("Unexpected output mismatch: %s", d)
	}
}

func testingHTTPClient(handler http.Handler) (*http.Client, func()) {
	s := httptest.NewTLSServer(handler)

	cli := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, s.Listener.Addr().String())
			},
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	return cli, s.Close
}
