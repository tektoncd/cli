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
		deleteFunc:  successfulDeleteFunc,
	}, {
		description: "prints success message if names provided",
		names:       []string{"foo", "bar"},
		expectedOut: "FooBars deleted: \"foo\", \"bar\"\n",
		expectedErr: "",
		deleteFunc:  successfulDeleteFunc,
	}, {
		description: "prints errors if returned during delete",
		names:       []string{"baz"},
		expectedOut: "",
		expectedErr: "failed to delete foobar \"baz\": unsuccessful delete func\n",
		deleteFunc:  unsuccessfulDeleteFunc,
	}, {
		description: "prints multiple errors if multiple deletions fail",
		names:       []string{"baz", "quux"},
		expectedOut: "",
		expectedErr: "failed to delete foobar \"baz\": unsuccessful delete func\nfailed to delete foobar \"quux\": unsuccessful delete func\n",
		deleteFunc:  unsuccessfulDeleteFunc,
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

func TestWithRelated(t *testing.T) {
	type related struct {
		kind   string
		list   func(string) ([]string, error)
		delete func(string) error
	}

	for _, tc := range []struct {
		description string
		related     []related
		expectedErr string
	}{{
		description: "single related kind is deleted ok",
		related: []related{{
			kind:   "FooBar",
			list:   successfulListFunc,
			delete: successfulDeleteFunc,
		}},
	}, {
		description: "multiple related kinds are deleted ok",
		related: []related{{
			kind:   "FooBar",
			list:   successfulListFunc,
			delete: successfulDeleteFunc,
		}, {
			kind:   "BazQuux",
			list:   successfulListFunc,
			delete: successfulDeleteFunc,
		}},
	}, {
		description: "failure to delete one related kind does not cause others to fail",
		related: []related{{
			kind:   "FooBar",
			list:   successfulListFunc,
			delete: unsuccessfulDeleteFunc,
		}, {
			kind:   "BazQuux",
			list:   successfulListFunc,
			delete: successfulDeleteFunc,
		}},
		expectedErr: `deleting relation of "foo", failed to delete foobar "related1": unsuccessful delete func
deleting relation of "foo", failed to delete foobar "related2": unsuccessful delete func
`,
	}, {
		description: "failure to list one related kind does not cause others to fail deletion",
		related: []related{{
			kind:   "FooBar",
			list:   unsuccessfulListFunc,
			delete: successfulDeleteFunc,
		}, {
			kind:   "BazQuux",
			list:   successfulListFunc,
			delete: successfulDeleteFunc,
		}},
		expectedErr: "fetching list of foobars related to \"foo\" failed: unsuccessful list func\n",
	}} {
		names := []string{"foo"}

		t.Run(tc.description, func(t *testing.T) {
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			streams := &cli.Stream{Out: stdout, Err: stderr}
			d := New("FooBar", successfulDeleteFunc)
			for _, related := range tc.related {
				d.WithRelated(related.kind, related.list, related.delete)
			}
			if err := d.Execute(streams, names); err != nil {
				if tc.expectedErr == "" {
					t.Errorf("unexpected error: %v", err)
				}
			}
			if stderr.String() != tc.expectedErr {
				t.Errorf("expected stderr %q received %q", tc.expectedErr, stderr.String())
			}
		})
	}
}

func successfulListFunc(string) ([]string, error) {
	return []string{"related1", "related2"}, nil
}

func unsuccessfulListFunc(string) ([]string, error) {
	return nil, errors.New("unsuccessful list func")
}

func successfulDeleteFunc(string) error {
	return nil
}

func unsuccessfulDeleteFunc(string) error {
	return errors.New("unsuccessful delete func")
}
