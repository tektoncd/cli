package deleter

import (
	"strings"
	"testing"

	"github.com/tektoncd/cli/pkg/cli"
)

func TestDelete(t *testing.T) {
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
	},
	} {
		t.Run(tc.description, func(t *testing.T) {
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			streams := &cli.Stream{Out: stdout, Err: stderr}
			d := New("FooBar", tc.deleteFunc)
			d.Delete(tc.names)
			d.PrintSuccesses(streams)
			if err := d.Errors(); err != nil {
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

func TestDeleteRelated(t *testing.T) {
	for _, tc := range []struct {
		description string
		relatedKind string
		listFunc    func(string) ([]string, error)
		deleteFunc  func(string) error
		expectedOut string
		expectedErr string
	}{{
		description: "doesnt print anything if no related are configured",
	}, {
		description: "prints success message with deleted relations",
		relatedKind: "FooBarRun",
		listFunc:    successfulListFunc("fbr1", "fbr2"),
		deleteFunc:  successfulDeleteFunc(),
		expectedOut: "FooBarRuns deleted: \"fbr1\", \"fbr2\"\n",
	}} {
		t.Run(tc.description, func(t *testing.T) {
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			streams := &cli.Stream{Out: stdout, Err: stderr}
			d := New("FooBar", successfulDeleteFunc())
			if tc.relatedKind != "" {
				d.WithRelated(tc.relatedKind, tc.listFunc, tc.deleteFunc)
				d.DeleteRelated([]string{"foo"})
			}
			d.PrintSuccesses(streams)
			if err := d.Errors(); err != nil {
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

func TestDeleteAndDeleteRelated(t *testing.T) {
	for _, tc := range []struct {
		description       string
		kind              string
		names             []string
		relatedKind       string
		deleteFunc        func(string) error
		listRelatedFunc   func(string) ([]string, error)
		deleteRelatedFunc func(string) error
		expectedOut       string
		expectedErr       string
	}{{
		description: "doesnt print anything if no related are configured",
	}, {
		description:       "prints success message with deleted and deleted relations",
		kind:              "FooBar",
		names:             []string{"fb1"},
		relatedKind:       "FooBarRun",
		deleteFunc:        successfulDeleteFunc(),
		listRelatedFunc:   successfulListFunc("fbr1", "fbr2"),
		deleteRelatedFunc: successfulDeleteFunc(),
		expectedOut:       "FooBarRuns deleted: \"fbr1\", \"fbr2\"\nFooBars deleted: \"fb1\"\n",
	}} {
		t.Run(tc.description, func(t *testing.T) {
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			streams := &cli.Stream{Out: stdout, Err: stderr}
			d := New(tc.kind, tc.deleteFunc)
			d.WithRelated(tc.relatedKind, tc.listRelatedFunc, tc.deleteRelatedFunc)
			deletedNames := d.Delete(tc.names)
			d.DeleteRelated(deletedNames)
			d.PrintSuccesses(streams)
			if err := d.Errors(); err != nil {
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

func successfulDeleteFunc() func(string) error {
	return func(string) error {
		return nil
	}
}

func successfulListFunc(returnedNames ...string) func(string) ([]string, error) {
	return func(string) ([]string, error) {
		return returnedNames, nil
	}
}
