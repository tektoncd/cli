// Copyright © 2019 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package formatted

import (
	"bytes"
	"strconv"
	"testing"
	"text/template"

	"github.com/fatih/color"
	"github.com/tektoncd/cli/pkg/test"
	"gotest.tools/v3/assert"
)

func TestRainbowsColours(t *testing.T) {
	rb := newRainbow()
	assert.Equal(t, rb.counter.value, uint32(0)) // nothing

	_ = rb.get("a") // get a label
	assert.Equal(t, rb.counter.value, uint32(1))

	_ = rb.get("b") // incremented
	assert.Equal(t, rb.counter.value, uint32(2))

	_ = rb.get("a") // no increment (cached)
	assert.Equal(t, rb.counter.value, uint32(2))

	rb = newRainbow()
	for c := range palette {
		a := strconv.Itoa(c)
		rb.get(a)
	}
	assert.Equal(t, rb.counter.value, uint32(0)) // Looped back to 0
}

func TestNoDecoration(t *testing.T) {
	type args struct {
		colorString string
		message     string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test that no colour get passed when running in tests",
			args: args{"foo", "message"},
			want: "message",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DecorateAttr(tt.args.colorString, tt.args.message); got != tt.want {
				t.Errorf("DecorateAttr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecoration(t *testing.T) {
	// We disable emoji and other colourful stuff while testing,
	// but here we want to explicitly enable it.
	color.NoColor = false
	defer func() {
		color.NoColor = true
	}()

	funcMap := template.FuncMap{
		"decorate": DecorateAttr,
	}
	aTemplate := `{{decorate "timeouts" ""}}{{decorate "skippedtasks" ""}}{{decorate "bullet" "Foo"}} {{decorate "check" ""}}{{decorate "resources" ""}}{{decorate "params" ""}}{{decorate "results" ""}}{{decorate "workspaces" ""}}{{decorate "tasks" ""}}{{decorate "pipelineruns" ""}}{{decorate "status" ""}}{{decorate "inputresources" ""}}{{decorate "outputresources" ""}}{{decorate "steps" ""}}{{decorate "message" ""}}{{decorate "taskruns" ""}}{{decorate "sidecars" ""}}{{decorate "red" "Red"}} {{decorate "underline" "Foo"}}`
	processed := template.Must(template.New("Describe Pipeline").Funcs(funcMap).Parse(aTemplate))
	buf := new(bytes.Buffer)

	if err := processed.Execute(buf, nil); err != nil {
		t.Error("Could not process the template.")
	}
	test.AssertOutput(t, "⏱  ⏭️  ∙ Foo ✔ ️📦 ⚓ 📝 📂 🗒  ⛩  🌡️  📨 📡 🦶 💌 🗂  🚗 \x1b[91mRed\x1b[0m \x1b[4mFoo\x1b[24m", buf.String())

}
