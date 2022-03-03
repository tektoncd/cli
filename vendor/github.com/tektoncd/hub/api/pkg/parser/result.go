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

package parser

import (
	"fmt"
	"strings"
)

type IssueType int

const (
	Critical IssueType = iota
	Warning
	Info
)

func (t IssueType) String() string {
	return [...]string{"critical", "warning", "info"}[t]
}

type Issue struct {
	Type    IssueType
	Message string
}

type Result struct {
	Errors []error
	Issues []Issue
}

func (r *Result) add(t IssueType, format string, args ...interface{}) {
	if r.Issues == nil {
		r.Issues = []Issue{}
	}
	r.Issues = append(r.Issues, Issue{Type: t, Message: fmt.Sprintf(format, args...)})

}

func (r *Result) Critical(format string, args ...interface{}) {
	r.add(Critical, format, args...)
}

func (r *Result) Warn(format string, args ...interface{}) {
	r.add(Warning, format, args...)
}

func (r *Result) Info(format string, args ...interface{}) {
	r.add(Info, format, args...)
}

func (r *Result) Error() string {

	if r.Errors == nil {
		return ""
	}

	if len(r.Errors) == 1 {
		return r.Errors[0].Error()
	}

	buf := strings.Builder{}
	for _, err := range r.Errors {
		buf.WriteString(err.Error())
		buf.WriteString("\n")
	}
	return buf.String()
}

func (r *Result) AddError(err error) {
	if r.Errors == nil {
		r.Errors = []error{}
	}
	r.Errors = append(r.Errors, err)
}

func (r *Result) Combine(other Result) {
	r.Issues = append(r.Issues, other.Issues...)
	r.Errors = append(r.Errors, other.Errors...)
}
