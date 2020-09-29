package eventlistener

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggertest "github.com/tektoncd/triggers/test"
	"gotest.tools/v3/golden"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLogsEventListener(t *testing.T) {
	now := time.Now()

	els := []*v1alpha1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "eventlistener-no-pods",
			},
		},
	}

	tests := []struct {
		name       string
		command    *cobra.Command
		args       []string
		wantError  bool
		goldenFile bool
		want       string
	}{
		{
			name:       "No arguments passed",
			command:    commandLogs(t, els, now),
			args:       []string{"logs"},
			wantError:  true,
			goldenFile: false,
			want:       "accepts 1 arg(s), received 0",
		},
		{
			name:       "No EventListener found",
			command:    commandLogs(t, els, now),
			args:       []string{"logs", "notFound"},
			wantError:  true,
			goldenFile: false,
			want:       "failed to get EventListener notFound: eventlisteners.triggers.tekton.dev \"notFound\" not found",
		},
		{
			name:       "No EventListener pods",
			command:    commandLogs(t, els, now),
			args:       []string{"logs", "eventlistener-no-pods"},
			wantError:  false,
			goldenFile: false,
			want:       "No pods available for EventListener eventlistener-no-pods\n",
		},
		{
			name:       "Tail option as 0 results in error",
			command:    commandLogs(t, els, now),
			args:       []string{"logs", "eventlistener-no-pods", "-t", "0"},
			wantError:  true,
			goldenFile: false,
			want:       "tail cannot be 0 or less than 0 unless -1 for all pods",
		},
		{
			name:       "Tail option as -2 results in error",
			command:    commandLogs(t, els, now),
			args:       []string{"logs", "eventlistener-no-pods", "-t", "-2"},
			wantError:  true,
			goldenFile: false,
			want:       "tail cannot be 0 or less than 0 unless -1 for all pods",
		},
	}

	for _, td := range tests {
		t.Run(td.name, func(t *testing.T) {
			got, err := test.ExecuteCommand(td.command, td.args...)

			if err != nil && !td.wantError {
				t.Errorf("Unexpected error: %v", err)
			}
			if td.goldenFile {
				golden.Assert(t, got, strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
			} else {
				if err != nil {
					test.AssertOutput(t, td.want, err.Error())
				} else {
					test.AssertOutput(t, td.want, got)
				}
			}
		})
	}
}

func commandLogs(t *testing.T, els []*v1alpha1.EventListener, now time.Time) *cobra.Command {
	clock := clockwork.NewFakeClockAt(now)
	cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els})
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Triggers: cs.Triggers}
	return Command(p)
}
