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

package log //nolint:revive

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/formatted"
)

type fullWriter struct {
	io.Writer
}

func (w fullWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	if err == nil && n != len(p) {
		err = io.ErrShortWrite
	}
	return n, err
}

// Writer helps logging pod"s log
type Writer struct {
	fmt         *formatted.Color
	logType     string
	prefixing   bool
	displayName bool
	output      string
}

// NewWriter returns the new instance of LogWriter
func NewWriter(logType string, prefixing bool, output string) *Writer {
	return &Writer{
		fmt:       formatted.NewColor(),
		logType:   logType,
		prefixing: prefixing,
		output:    output,
	}
}

// WithDisplayName sets the display name
func (lw *Writer) WithDisplayName(displayName bool) *Writer {
	lw.displayName = displayName
	return lw
}

// Write formatted pod's logs

func (lw *Writer) Write(s *cli.Stream, logC <-chan Log, errC <-chan error) error {
	type jsonLogEntry struct {
		Pipeline  string `json:"pipeline,omitempty"`
		Task      string `json:"task,omitempty"`
		Step      string `json:"step,omitempty"`
		Timestamp string `json:"timestamp"`
		Message   string `json:"message"`
	}

	encoder := json.NewEncoder(fullWriter{Writer: s.Out})
	encoder.SetEscapeHTML(false) // Prevents escaping <, >, & in log messages

	var readerErr error

	for logC != nil || errC != nil {
		select {
		case l, ok := <-logC:
			if !ok {
				logC = nil
				continue
			}

			if l.EndOfStep {
				if lw.output != "json" {
					fmt.Fprintf(s.Out, "\n")
				}
				continue
			}

			if lw.displayName && l.TaskDisplayName != "" {
				l.Task = l.TaskDisplayName
			}

			if lw.output == "json" {
				entry := jsonLogEntry{
					Pipeline:  l.Pipeline,
					Task:      l.Task,
					Step:      l.Step,
					Timestamp: l.Timestamp.Format(time.RFC3339Nano),
					Message:   l.Log,
				}
				// Encode() automatically flushes and adds a newline (NDJSON)
				if err := encoder.Encode(entry); err != nil {
					return fmt.Errorf("failed to encode log entry: %w", err)
				}
			} else {
				// Existing human-readable logic
				if lw.prefixing {
					switch lw.logType {
					case LogTypePipeline:
						lw.fmt.Rainbow.Fprintf(l.Step, s.Out, "[%s : %s] ", l.Task, l.Step)
					case LogTypeTask:
						lw.fmt.Rainbow.Fprintf(l.Step, s.Out, "[%s] ", l.Step)
					}
				}

				if !l.Timestamp.IsZero() {
					fmt.Fprintf(s.Out, "%s ", l.Timestamp.Format(time.RFC3339Nano))
				}

				fmt.Fprintf(s.Out, "%s\n", l.Log)
			}
		case e, ok := <-errC:
			if !ok {
				errC = nil
				continue
			}

			if lw.output == "json" {
				if readerErr == nil {
					readerErr = e
				} else {
					readerErr = errors.Join(readerErr, e)
				}
				continue
			}
			lw.fmt.Error(s.Err, "%s\n", e)
		}
	}
	return readerErr
}
