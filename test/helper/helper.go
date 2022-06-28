// Copyright © 2020 The Tekton Authors.
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

package helper

import (
	"bytes"
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"text/template"

	"github.com/Netflix/go-expect"
	"github.com/creack/pty"
	"github.com/google/go-cmp/cmp"
	"github.com/hinshun/vt10x"
	gotestcmp "gotest.tools/assert/cmp"
)

func Process(t *template.Template, vars interface{}) string {
	var tmplBytes bytes.Buffer
	err := t.Execute(&tmplBytes, vars)
	if err != nil {
		panic(err)
	}
	return tmplBytes.String()
}

func ProcessString(str string, vars interface{}) string {
	tmpl, err := template.New("tmpl").Parse(str)
	if err != nil {
		panic(err)
	}
	return Process(tmpl, vars)
}

func dir() string {
	_, b, _, _ := runtime.Caller(0)
	return path.Join(path.Dir(b), "..", "resources")
}

func GetResourcePath(elem ...string) string {
	tmp := dir()
	path := append([]string{tmp}, elem...)
	return filepath.Join(path...)
}

func AssertOutput(t *testing.T, expected, actual interface{}) {
	t.Helper()
	diff := cmp.Diff(actual, expected)
	if diff == "" {
		return
	}

	t.Errorf(`
Unexpected output:
%s

Expected
%s

Actual
%s
`, diff, expected, actual)
}

// ContainsAll is a comparison utility, compares given substrings against
// target string and returns the gotest.tools/assert/cmp.Comparison function.
// Provide target string as first arg, followed by any number of substring as args
func ContainsAll(target string, substrings ...string) gotestcmp.Comparison {
	return func() gotestcmp.Result {
		var missing []string
		for _, sub := range substrings {
			if !strings.Contains(target, sub) {
				missing = append(missing, sub)
			}
		}
		if len(missing) > 0 {
			return gotestcmp.ResultFailure(fmt.Sprintf("\nActual output: %s\nMissing strings: %s", target, strings.Join(missing, ", ")))
		}
		return gotestcmp.ResultSuccess
	}
}

// NewVT10XConsole returns a new expect.Console that multiplexes the
// Stdin/Stdout to a VT10X terminal, allowing Console to interact with an
// application sending ANSI escape sequences.
func NewVT10XConsole(opts ...expect.ConsoleOpt) (*expect.Console, vt10x.Terminal, error) {
	ptm, pts, err := pty.Open()
	if err != nil {
		return nil, nil, err
	}

	term := vt10x.New(vt10x.WithWriter(pts))
	c, err := expect.NewConsole(append(opts, expect.WithStdin(ptm), expect.WithStdout(term), expect.WithCloser(ptm, pts))...)
	if err != nil {
		return nil, nil, err
	}

	return c, term, nil
}
