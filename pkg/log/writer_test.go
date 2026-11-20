// Copyright © 2026 The Tekton Authors.
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
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/tektoncd/cli/pkg/cli"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

func TestWriterReturnsJSONEncodingError(t *testing.T) {
	logs := make(chan Log, 1)
	logs <- Log{Task: "task", Step: "step", Log: "message"}
	close(logs)

	err := NewWriter(LogTypeTask, false, "json").Write(&cli.Stream{
		Out: failingWriter{},
		Err: failingWriter{},
	}, logs, nil)
	if err == nil {
		t.Fatal("expected encoding error")
	}
	if !strings.Contains(err.Error(), "write failed") {
		t.Fatalf("error = %v, want underlying writer error", err)
	}
}

func TestWriterReturnsShortWrite(t *testing.T) {
	logs := make(chan Log, 1)
	logs <- Log{Task: "task", Step: "step", Log: "message"}
	close(logs)

	err := NewWriter(LogTypeTask, false, "json").Write(&cli.Stream{Out: shortWriter{}}, logs, nil)
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("error = %v, want io.ErrShortWrite", err)
	}
}

func TestWriterWritesJSONRecord(t *testing.T) {
	logs := make(chan Log, 1)
	logs <- Log{Task: "task", Step: "step", Log: "message"}
	close(logs)
	var out bytes.Buffer

	if err := NewWriter(LogTypeTask, false, "json").Write(&cli.Stream{Out: &out}, logs, nil); err != nil {
		t.Fatalf("Write() = %v", err)
	}
	if got, want := out.String(), "{\"task\":\"task\",\"step\":\"step\",\"timestamp\":\"0001-01-01T00:00:00Z\",\"message\":\"message\"}\n"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestWriterReturnsJSONLogError(t *testing.T) {
	errs := make(chan error, 1)
	errs <- errors.New("log stream failed")
	close(errs)

	err := NewWriter(LogTypeTask, false, "json").Write(&cli.Stream{
		Out: new(bytes.Buffer),
		Err: new(bytes.Buffer),
	}, nil, errs)
	if err == nil {
		t.Fatal("expected log stream error")
	}
	if err.Error() != "log stream failed" {
		t.Fatalf("error = %q, want %q", err.Error(), "log stream failed")
	}
}

func TestWriterDrainsJSONLogsBeforeReturningReaderError(t *testing.T) {
	logs := make(chan Log, 2)
	logs <- Log{Task: "task", Step: "first", Log: "before"}
	logs <- Log{Task: "task", Step: "second", Log: "after"}
	close(logs)

	errs := make(chan error, 2)
	errs <- errors.New("first reader error")
	errs <- errors.New("second reader error")
	close(errs)

	var out bytes.Buffer
	err := NewWriter(LogTypeTask, false, "json").Write(&cli.Stream{
		Out: &out,
		Err: new(bytes.Buffer),
	}, logs, errs)
	if err == nil || err.Error() != "first reader error" {
		t.Fatalf("error = %v, want first reader error", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d JSON records, want 2: %q", len(lines), out.String())
	}
	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("invalid JSON record %q: %v", line, err)
		}
	}
}
