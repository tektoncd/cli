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

func TestWriterReturnsReaderError(t *testing.T) {
	logs := make(chan Log)
	errs := make(chan error, 2)

	readerErr1 := errors.New("failed to fetch pod logs")
	readerErr2 := errors.New("container failed")
	errs <- readerErr1
	errs <- readerErr2
	close(errs)
	close(logs)

	var stderr bytes.Buffer
	err := NewWriter(LogTypeTask, false, "json").Write(&cli.Stream{
		Out: &bytes.Buffer{},
		Err: &stderr,
	}, logs, errs)

	if !errors.Is(err, readerErr1) || !errors.Is(err, readerErr2) {
		t.Fatalf("Write() error = %v, want errors containing %v and %v", err, readerErr1, readerErr2)
	}
	if stderr.Len() > 0 {
		t.Errorf("stderr = %q, expected no plain-text stderr output in JSON mode", stderr.String())
	}
}

func TestWriterHumanReadablePrintsErrors(t *testing.T) {
	logs := make(chan Log)
	errs := make(chan error, 1)

	readerErr := errors.New("failed to fetch pod logs")
	errs <- readerErr
	close(errs)
	close(logs)

	var stderr bytes.Buffer
	err := NewWriter(LogTypeTask, false, "").Write(&cli.Stream{
		Out: &bytes.Buffer{},
		Err: &stderr,
	}, logs, errs)

	if err != nil {
		t.Fatalf("expected nil error in human-readable mode, got %v", err)
	}
	if !strings.Contains(stderr.String(), readerErr.Error()) {
		t.Errorf("stderr = %q, want it to contain %q", stderr.String(), readerErr)
	}
}
