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
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"text/template"

	"github.com/google/go-cmp/cmp"
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
