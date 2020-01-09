package deleter

import (
	"errors"
	"strings"
	"testing"

	"github.com/tektoncd/cli/pkg/cli"
)

func TestExecute(t *testing.T) {
	for _, tc := range []struct {
		description string
		names       []string
		expectedOut string
		expectedErr string
		deleteFunc  func(string) error
	}{{
		description: "doesnt print anything if no names provided",
		names:       []string{},
		expectedOut: "",
		expectedErr: "",
		deleteFunc:  func(string) error { return nil },
	}, {
		description: "prints success message if names provided",
		names:       []string{"foo", "bar"},
		expectedOut: "FooBars deleted: \"foo\", \"bar\"\n",
		expectedErr: "",
		deleteFunc:  func(string) error { return nil },
	}, {
		description: "prints errors if returned during delete",
		names:       []string{"baz"},
		expectedOut: "",
		expectedErr: "failed to delete foobar \"baz\": there was an unfortunate incident\n",
		deleteFunc:  func(string) error { return errors.New("there was an unfortunate incident") },
	}, {
		description: "prints multiple errors if multiple deletions fail",
		names:       []string{"baz", "quux"},
		expectedOut: "",
		expectedErr: "failed to delete foobar \"baz\": there was an unfortunate incident\nfailed to delete foobar \"quux\": there was an unfortunate incident\n",
		deleteFunc:  func(string) error { return errors.New("there was an unfortunate incident") },
	}} {
		t.Run(tc.description, func(t *testing.T) {
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			streams := &cli.Stream{Out: stdout, Err: stderr}
			d := New("FooBar", tc.deleteFunc)
			if err := d.Execute(streams, tc.names); err != nil {
				if tc.expectedErr == "" {
					t.Errorf("unexpected error: %v", err)
				}
			}
			if stdout.String() != tc.expectedOut {
				t.Errorf("expected stdout %q received %q", tc.expectedOut, stdout.String())
			}
			if stderr.String() != tc.expectedErr {
				t.Errorf("expected stderr %q received %q", tc.expectedErr, stderr.String())
			}
		})
	}
}
