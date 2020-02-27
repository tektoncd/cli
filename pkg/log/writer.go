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

package log

import (
	"fmt"

	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
)

// Writer helps logging pod"s log
type Writer struct {
	fmt     *formatted.Color
	logType string
}

// NewWriter returns the new instance of LogWriter
func NewWriter(logType string) *Writer {
	return &Writer{
		fmt:     formatted.NewColor(),
		logType: logType,
	}
}

// Write formatted pod's logs
func (lw *Writer) Write(s *cli.Stream, logC <-chan Log, errC <-chan error) {
	for logC != nil || errC != nil {
		select {
		case l, ok := <-logC:
			if !ok {
				logC = nil
				continue
			}

			if l.Log == "EOFLOG" {
				fmt.Fprintf(s.Out, "\n")
				continue
			}

			switch lw.logType {
			case LogTypePipeline:
				lw.fmt.Rainbow.Fprintf(l.Step, s.Out, "[%s : %s] ", l.Task, l.Step)
			case LogTypeTask:
				lw.fmt.Rainbow.Fprintf(l.Step, s.Out, "[%s] ", l.Step)
			}

			fmt.Fprintf(s.Out, "%s\n", l.Log)
		case e, ok := <-errC:
			if !ok {
				errC = nil
				continue
			}
			lw.fmt.Error(s.Err, "%s\n", e)
		}
	}
}
