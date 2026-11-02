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

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tektoncd/cli/pkg/exitcode"
	"github.com/tektoncd/cli/pkg/test"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
	"gotest.tools/v3/golden"
)

func TestCommand_no_global_flags(t *testing.T) {
	unwantedflag := "OPTION-SHOULD-NOT-BE-HERE"
	pflag.String(unwantedflag, "", "An option that we really don't want to show")

	p := &test.Params{}
	pipelinerun := Root(p)
	out, err := test.ExecuteCommand(pipelinerun)
	if err != nil {
		t.Errorf("TestCommand_no_global_flags should not result in an error but did: %v", err)
	}

	if strings.Contains(out, unwantedflag) {
		t.Errorf("The Flag: %s, should not have been added to the global flags", unwantedflag)
	}
}

func TestPrintError_plain(t *testing.T) {
	var buf bytes.Buffer

	printError(fmt.Errorf("boom"), &buf, false)

	if got, want := buf.String(), "Error: boom\n"; got != want {
		t.Errorf("printError plain output = %q, want %q", got, want)
	}
}

func TestPrintError_json(t *testing.T) {
	var buf bytes.Buffer

	printError(exitcode.New(exitcode.NotFound, "thing not found"), &buf, true)

	var payload struct {
		Error string `json:"error"`
		Code  int    `json:"code"`
	}
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("printError json output is not valid JSON: %v (%q)", err, buf.String())
	}
	if payload.Error != "thing not found" {
		t.Errorf("error = %q, want %q", payload.Error, "thing not found")
	}
	if payload.Code != exitcode.NotFound {
		t.Errorf("code = %d, want %d", payload.Code, exitcode.NotFound)
	}
}

func TestOutputFromArgs(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		want      string
		wantFound bool
	}{
		{"space form", []string{"list", "--output", "json"}, "json", true},
		{"equals form", []string{"list", "--output=json"}, "json", true},
		{"short space", []string{"list", "-o", "yaml"}, "yaml", true},
		{"short equals", []string{"list", "-o=json"}, "json", true},
		{"short joined", []string{"list", "-ojson"}, "json", true},
		{"uppercase", []string{"list", "--output", "JSON"}, "JSON", true},
		{"after invalid flag", []string{"list", "--limit=abc", "--output", "json"}, "json", true},
		{"absent", []string{"list", "--limit=abc"}, "", false},
		{"trailing without value", []string{"list", "--output"}, "", false},
		{"after terminator", []string{"list", "--", "--output", "json"}, "", false},
		{"value consumed by long flag", []string{"list", "--component", "-ojson"}, "", false},
		{"value consumed by short flag", []string{"list", "-n", "-ojson"}, "", false},
		{"output after consumed value", []string{"list", "--component", "x", "-ojson"}, "json", true},
		{"last long wins", []string{"list", "--output", "yaml", "--output", "json"}, "json", true},
		{"last after invalid flag", []string{"list", "--limit=abc", "--output", "yaml", "--output", "json"}, "json", true},
		{"last short wins", []string{"list", "-o", "yaml", "-ojson"}, "json", true},
		{"override before terminator", []string{"list", "--output", "yaml", "--", "--output", "json"}, "yaml", true},
		{"bool short equals value not scanned", []string{"list", "-o", "json", "-C=oops"}, "json", true},
		{"bool short equals value absent output", []string{"list", "-C=oops"}, "", false},
		{"output after bool short equals", []string{"list", "-C=oops", "-ojson"}, "json", true},
		{"unknown shorthand in cluster keeps earlier", []string{"list", "-o", "json", "-zoyaml"}, "json", true},
		{"trailing valueless short keeps earlier", []string{"list", "-o", "json", "-o"}, "json", true},
		{"trailing valueless long keeps earlier", []string{"list", "--output", "json", "--output"}, "json", true},
	}
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.StringP("output", "o", "", "")
	flags.String("limit", "", "")
	flags.String("component", "", "")
	flags.StringP("namespace", "n", "", "")
	flags.BoolP("cluster", "C", false, "")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, found := outputFromArgs(flags, tc.args)
			if got != tc.want || found != tc.wantFound {
				t.Errorf("outputFromArgs(%v) = (%q, %t), want (%q, %t)", tc.args, got, found, tc.want, tc.wantFound)
			}
		})
	}
}

// TestWantsJSONError covers how the JSON error format is resolved from the raw
// arguments, independently of successful flag parsing.  Cobra stops at the
// first invalid flag, so a later --output may never be stored; the raw
// arguments are therefore the source of truth and pflag's last-occurrence
// semantics must be honored.
func TestWantsJSONError(t *testing.T) {
	cases := []struct {
		name       string
		outDefault string
		args       []string
		want       bool
	}{
		{"flag set to json", "", []string{"list", "--output", "json"}, true},
		{"uppercase json", "", []string{"list", "--output", "JSON"}, true},
		{"unparsed after invalid flag", "", []string{"list", "--limit=abc", "--output", "json"}, true},
		{"non-empty default with json", "table", []string{"list", "--limit=abc", "--output", "json"}, true},
		{"non-empty default retained", "table", []string{"list", "--limit=abc"}, false},
		{"last output wins", "", []string{"list", "--output", "yaml", "--output", "json"}, true},
		{"yaml not json", "", []string{"list", "--output", "yaml"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "demo"}
			cmd.Flags().StringP("output", "o", tc.outDefault, "")
			if got := wantsJSONError(cmd, tc.args); got != tc.want {
				t.Errorf("wantsJSONError(%v) = %t, want %t", tc.args, got, tc.want)
			}
		})
	}
}

// output is valid JSON even for a deprecated command, whose deprecation warning
// Cobra prints before flag parsing.  The warning must not corrupt the JSON.
func TestExecute_json_stderr_valid(t *testing.T) {
	p := &test.Params{}
	var errBuf bytes.Buffer

	code := Execute(Root(p), []string{"hub", "search", "--limit=abc", "--output", "json"}, &errBuf)

	if code != exitcode.InvalidInput {
		t.Fatalf("exit code = %d, want %d (stderr=%q)", code, exitcode.InvalidInput, errBuf.String())
	}
	var payload struct {
		Error string `json:"error"`
		Code  int    `json:"code"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(errBuf.Bytes()), &payload); err != nil {
		t.Fatalf("stderr is not valid JSON: %v (%q)", err, errBuf.String())
	}
	if payload.Code != exitcode.InvalidInput {
		t.Errorf("payload code = %d, want %d", payload.Code, exitcode.InvalidInput)
	}
}

// TestExecute_last_output_wins verifies that a later --output overrides an
// earlier one even when parsing stopped at an invalid flag in between, matching
// pflag's last-occurrence semantics.
func TestExecute_last_output_wins(t *testing.T) {
	p := &test.Params{}
	var errBuf bytes.Buffer

	code := Execute(Root(p), []string{"pipelinerun", "list", "--limit=abc", "--output", "yaml", "--output", "json"}, &errBuf)

	if code != exitcode.InvalidInput {
		t.Fatalf("exit code = %d, want %d", code, exitcode.InvalidInput)
	}
	if !json.Valid(bytes.TrimSpace(errBuf.Bytes())) {
		t.Errorf("stderr is not valid JSON: %q", errBuf.String())
	}
}

// TestExecute_plain_keeps_deprecation verifies that outside JSON mode the
// executed command's deprecation notice is preserved (Execute only suppresses
// it for JSON output, where a plain-text preamble would invalidate the JSON on
// stderr).
func TestExecute_plain_keeps_deprecation(t *testing.T) {
	p := &test.Params{}
	root := Root(p)
	var errBuf bytes.Buffer

	code := Execute(root, []string{"hub", "search", "--limit=abc"}, &errBuf)

	if code != exitcode.InvalidInput {
		t.Fatalf("exit code = %d, want %d", code, exitcode.InvalidInput)
	}
	target, _, err := root.Find([]string{"hub", "search", "--limit=abc"})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if target.Deprecated == "" {
		t.Errorf("plain mode should keep the deprecation notice, but it was cleared")
	}
}

// TestExecute_json_suppresses_deprecation verifies that JSON error mode clears
// the executed command's deprecation notice so Cobra does not emit a plain-text
// preamble to stderr.
func TestExecute_json_suppresses_deprecation(t *testing.T) {
	p := &test.Params{}
	root := Root(p)
	var errBuf bytes.Buffer

	Execute(root, []string{"hub", "search", "--limit=abc", "--output", "json"}, &errBuf)

	target, _, err := root.Find([]string{"hub", "search", "--limit=abc", "--output", "json"})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if target.Deprecated != "" {
		t.Errorf("JSON mode should suppress the deprecation notice, got %q", target.Deprecated)
	}
}

// TestSuppressWarnings verifies that suppressWarnings clears the command and
// flag deprecation notices (written to the out writer) and discards the error
// writer that hook warnings use, so none of them can leak before a JSON error.
func TestSuppressWarnings(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)

	sub := &cobra.Command{
		Use:        "sub",
		Deprecated: "use something else",
		RunE:       func(_ *cobra.Command, _ []string) error { return nil },
	}
	sub.Flags().String("old", "", "")
	if err := sub.Flags().MarkDeprecated("old", "use --new"); err != nil {
		t.Fatalf("mark deprecated: %v", err)
	}
	root.AddCommand(sub)

	target, _, err := root.Find([]string{"sub"})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	suppressWarnings(target)

	if target.Deprecated != "" {
		t.Errorf("command deprecation notice not cleared, got %q", target.Deprecated)
	}
	target.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Deprecated != "" {
			t.Errorf("flag %q deprecation notice not cleared, got %q", f.Name, f.Deprecated)
		}
	})
	// A hook warning written to the error writer must be discarded.
	fmt.Fprintln(target.ErrOrStderr(), "WARNING: deprecated hook")
	if errBuf.Len() != 0 {
		t.Errorf("error writer not discarded, got %q", errBuf.String())
	}
}

// TestExecute_json_suppresses_flag_deprecation verifies that a deprecated flag's
// warning does not leak to the shared out stream in JSON error mode, keeping
// stderr valid JSON.  pipeline start marks --timeout deprecated.
func TestExecute_json_suppresses_flag_deprecation(t *testing.T) {
	p := &test.Params{}
	root := Root(p)
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)

	args := []string{"pipeline", "start", "--timeout", "1s", "--output", "json", "--showlog"}
	code := Execute(root, args, &errBuf)

	if code != exitcode.InvalidInput {
		t.Fatalf("exit code = %d, want %d (stderr=%q)", code, exitcode.InvalidInput, errBuf.String())
	}
	if strings.Contains(strings.ToLower(outBuf.String()), "deprecated") {
		t.Errorf("deprecated-flag warning leaked to out stream: %q", outBuf.String())
	}
	if !json.Valid(bytes.TrimSpace(errBuf.Bytes())) {
		t.Errorf("stderr is not valid JSON: %q", errBuf.String())
	}
}

// TestExecute_json_suppresses_experimental_warning verifies that an
// experimental command's advisory warning (written via ErrOrStderr) is
// suppressed in JSON error mode so stderr stays valid JSON.  bundle is an
// experimental command, and an invalid kind fails with InvalidInput.
func TestExecute_json_suppresses_experimental_warning(t *testing.T) {
	p := &test.Params{}
	var errBuf bytes.Buffer

	args := []string{"bundle", "list", "example.com/repo:latest", "invalid-kind", "--output", "json"}
	code := Execute(Root(p), args, &errBuf)

	if code != exitcode.InvalidInput {
		t.Fatalf("exit code = %d, want %d (stderr=%q)", code, exitcode.InvalidInput, errBuf.String())
	}
	if strings.Contains(errBuf.String(), "experimental") {
		t.Errorf("experimental warning leaked to stderr: %q", errBuf.String())
	}
	if !json.Valid(bytes.TrimSpace(errBuf.Bytes())) {
		t.Errorf("stderr is not valid JSON: %q", errBuf.String())
	}
}

// TestExecute_logs_too_many_args verifies that exceeding a logs command's
// positional-argument limit is classified as InvalidInput (3) by the offline
// Args validator instead of reaching initOpts and exiting with the generic 1.
func TestExecute_logs_too_many_args(t *testing.T) {
	p := &test.Params{}
	for _, args := range [][]string{
		{"pipeline", "logs", "a", "b", "c"},
		{"task", "logs", "a", "b", "c"},
	} {
		var errBuf bytes.Buffer
		code := Execute(Root(p), args, &errBuf)
		if code != exitcode.InvalidInput {
			t.Errorf("Execute(%v) = %d, want %d (stderr=%q)", args, code, exitcode.InvalidInput, errBuf.String())
		}
	}
}

// TestOutputFromArgs_noOutputShorthand verifies that when a command registers
// --output without the -o shorthand (for example "task start"), a stray -o is
// not mistaken for the output flag and cannot override a valid --output value.
func TestOutputFromArgs_noOutputShorthand(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("output", "", "") // no shorthand
	cases := []struct {
		name      string
		args      []string
		want      string
		wantFound bool
	}{
		{"stray short does not override long", []string{"start", "task", "--output", "json", "-o", "yaml"}, "json", true},
		{"stray joined short ignored", []string{"start", "task", "-oyaml"}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, found := outputFromArgs(flags, tc.args)
			if got != tc.want || found != tc.wantFound {
				t.Errorf("outputFromArgs(%v) = (%q, %t), want (%q, %t)", tc.args, got, found, tc.want, tc.wantFound)
			}
		})
	}
}

// TestExitCode_classification verifies that client-side input errors surfaced
// by Cobra — invalid flag values, unknown flags and missing positional
// arguments — are classified as InvalidInput so tkn exits with the documented
// code instead of the generic error code.
func TestExitCode_classification(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"invalid flag value", []string{"pipelinerun", "list", "--limit=abc"}},
		{"unknown flag", []string{"pipelinerun", "list", "--does-not-exist"}},
		{"missing positional argument", []string{"pipelinerun", "cancel"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &test.Params{}
			_, err := test.ExecuteCommand(Root(p), tc.args...)
			if err == nil {
				t.Fatalf("expected an error for args %v", tc.args)
			}
			if got := exitcode.CodeFrom(err); got != exitcode.InvalidInput {
				t.Errorf("CodeFrom(%v) = %d, want %d (%v)", tc.args, got, exitcode.InvalidInput, err)
			}
		})
	}
}

// TestClassifyArgErrors_preservesClassified verifies that classifyArgErrors
// converts plain validation failures to InvalidInput but preserves errors that
// are already classified (for example NotFound returned by an Args validator
// that performs an API lookup).
func TestClassifyArgErrors_preservesClassified(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"plain validation error", errors.New("requires a name"), exitcode.InvalidInput},
		{"classified not found", exitcode.New(exitcode.NotFound, "not found"), exitcode.NotFound},
		{"classified unauthorized", exitcode.New(exitcode.Unauthorized, "forbidden"), exitcode.Unauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "demo"}
			cmd.Args = func(*cobra.Command, []string) error { return tc.err }
			classifyArgErrors(cmd)
			err := cmd.Args(cmd, nil)
			if got := exitcode.CodeFrom(err); got != tc.want {
				t.Errorf("CodeFrom(%v) = %d, want %d", err, got, tc.want)
			}
		})
	}
}

func TestCommand_suggest(t *testing.T) {
	p := &test.Params{}
	pipelinerun := Root(p)
	out, err := test.ExecuteCommand(pipelinerun, "pi")
	if err == nil {
		t.Errorf("No errors was defined. Output: %s", out)
	}
	expected := "unknown command \"pi\" for \"tkn\"\n\nDid you mean this?\n\tpipeline\n\tpipelinerun\n"
	test.AssertOutput(t, expected, err.Error())
}

func TestSubCommand_suggest(t *testing.T) {
	p := &test.Params{}
	pipelinedescriberun := Root(p)
	out, err := test.ExecuteCommand(pipelinedescriberun, "pipeline", "des")
	if err == nil {
		t.Errorf("No errors was defined. Output: %s", out)
	}
	expected := "unknown command \"des\" for \"tkn pipeline\"\n\nDid you mean this?\n\tdescribe\n"
	test.AssertOutput(t, expected, err.Error())
}

func TestPluginList(t *testing.T) {
	nd := fs.NewDir(t, "TestPluginList")
	defer nd.Remove()
	// nolint: gosec
	err := os.WriteFile(nd.Join("tkn-exec"), []byte("exec"), 0o700)
	assert.NilError(t, err)
	err = os.WriteFile(nd.Join("tkn-nonexec"), []byte("nonexec"), 0o600)
	assert.NilError(t, err)
	t.Setenv("PATH", nd.Path()+":/non/existing/path")
	t.Setenv("TKN_PLUGINS_DIR", "/non/existing/path")
	p := &test.Params{}
	cmd := Root(p)
	out, err := test.ExecuteCommand(cmd, "help")
	assert.NilError(t, err)
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}
