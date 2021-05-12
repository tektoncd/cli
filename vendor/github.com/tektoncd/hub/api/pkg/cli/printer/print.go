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

package printer

import (
	"fmt"
	"io"
	"text/tabwriter"
	"text/template"

	"github.com/tektoncd/hub/api/pkg/cli/formatter"
)

type Printer struct {
	out io.Writer
}

// New returns new object of Printer
func New(out io.Writer) *Printer {
	return &Printer{out: out}
}

// JSON prints formats json data and prints it
func (p *Printer) JSON(data []byte, err error) error {
	if err != nil {
		return err
	}

	res, err := formatter.FormatJSON(data)
	if err != nil {
		fmt.Fprintf(p.out, "ERROR: %s\n", err)
	}
	fmt.Fprint(p.out, string(res))
	return nil
}

// Tabbed prints data in table form based on the template passed
func (p *Printer) Tabbed(tmpl *template.Template, templateData interface{}) error {

	w := tabwriter.NewWriter(p.out, 0, 5, 3, ' ', tabwriter.TabIndent)
	defer w.Flush()

	return tmpl.Execute(w, templateData)
}

// Raw prints the raw byte array to printer's output stream
func (p *Printer) Raw(data []byte, err error) error {
	if err != nil {
		return err
	}

	fmt.Fprintln(p.out, string(data))
	return nil
}

// String prints the string to printer's output stream
func (p *Printer) String(str string) error {
	fmt.Fprintln(p.out, string(str))
	return nil
}
