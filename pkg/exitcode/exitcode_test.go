// Copyright © 2024 The Tekton Authors.
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

package exitcode_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/tektoncd/cli/pkg/exitcode"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestCodeFrom_nil(t *testing.T) {
	if got := exitcode.CodeFrom(nil); got != exitcode.Success {
		t.Errorf("CodeFrom(nil) = %d, want %d", got, exitcode.Success)
	}
}

func TestCodeFrom_plain_error(t *testing.T) {
	if got := exitcode.CodeFrom(fmt.Errorf("boom")); got != exitcode.GeneralError {
		t.Errorf("CodeFrom(plain) = %d, want %d", got, exitcode.GeneralError)
	}
}

func TestCodeFrom_exitcode_error(t *testing.T) {
	err := exitcode.New(exitcode.NotFound, "thing not found")
	if got := exitcode.CodeFrom(err); got != exitcode.NotFound {
		t.Errorf("CodeFrom(Error{NotFound}) = %d, want %d", got, exitcode.NotFound)
	}
}

func TestCodeFrom_wrapped_exitcode_error(t *testing.T) {
	base := exitcode.New(exitcode.Unauthorized, "forbidden")
	wrapped := fmt.Errorf("outer: %w", base)
	if got := exitcode.CodeFrom(wrapped); got != exitcode.Unauthorized {
		t.Errorf("CodeFrom(wrapped Unauthorized) = %d, want %d", got, exitcode.Unauthorized)
	}
}

func TestFromAPIError_nil(t *testing.T) {
	if err := exitcode.FromAPIError(nil); err != nil {
		t.Errorf("FromAPIError(nil) = %v, want nil", err)
	}
}

func TestFromAPIError_notFound(t *testing.T) {
	gr := schema.GroupResource{Group: "tekton.dev", Resource: "pipelineruns"}
	k8sErr := k8serrors.NewNotFound(gr, "my-pr")
	err := exitcode.FromAPIError(k8sErr)
	var e *exitcode.Error
	if !errors.As(err, &e) {
		t.Fatal("expected exitcode.Error")
	}
	if e.Code != exitcode.NotFound {
		t.Errorf("Code = %d, want %d", e.Code, exitcode.NotFound)
	}
}

func TestFromAPIError_forbidden(t *testing.T) {
	gr := schema.GroupResource{Group: "tekton.dev", Resource: "tasks"}
	k8sErr := k8serrors.NewForbidden(gr, "my-task", fmt.Errorf("forbidden"))
	err := exitcode.FromAPIError(k8sErr)
	var e *exitcode.Error
	if !errors.As(err, &e) {
		t.Fatal("expected exitcode.Error")
	}
	if e.Code != exitcode.Unauthorized {
		t.Errorf("Code = %d, want %d", e.Code, exitcode.Unauthorized)
	}
}

func TestFromAPIError_generic(t *testing.T) {
	plain := fmt.Errorf("connection refused")
	if got := exitcode.FromAPIError(plain); got != plain {
		t.Errorf("FromAPIError(generic) changed the error, want original")
	}
}

func TestFromAPIError_invalid(t *testing.T) {
	gk := schema.GroupKind{Group: "tekton.dev", Kind: "Pipeline"}
	k8sErr := k8serrors.NewInvalid(gk, "my-pipeline", nil)
	err := exitcode.FromAPIError(k8sErr)
	var e *exitcode.Error
	if !errors.As(err, &e) {
		t.Fatal("expected exitcode.Error")
	}
	if e.Code != exitcode.InvalidInput {
		t.Errorf("Code = %d, want %d", e.Code, exitcode.InvalidInput)
	}
}

func TestFromAPIError_badRequest(t *testing.T) {
	err := exitcode.FromAPIError(k8serrors.NewBadRequest("bad input"))
	var e *exitcode.Error
	if !errors.As(err, &e) {
		t.Fatal("expected exitcode.Error")
	}
	if e.Code != exitcode.InvalidInput {
		t.Errorf("Code = %d, want %d", e.Code, exitcode.InvalidInput)
	}
}

func TestFromAPIError_serverTimeout(t *testing.T) {
	gr := schema.GroupResource{Group: "tekton.dev", Resource: "pipelineruns"}
	err := exitcode.FromAPIError(k8serrors.NewServerTimeout(gr, "list", 1))
	var e *exitcode.Error
	if !errors.As(err, &e) {
		t.Fatal("expected exitcode.Error")
	}
	if e.Code != exitcode.Timeout {
		t.Errorf("Code = %d, want %d", e.Code, exitcode.Timeout)
	}
}

func TestFromAPIError_deadlineExceeded(t *testing.T) {
	err := exitcode.FromAPIError(context.DeadlineExceeded)
	var e *exitcode.Error
	if !errors.As(err, &e) {
		t.Fatal("expected exitcode.Error")
	}
	if e.Code != exitcode.Timeout {
		t.Errorf("Code = %d, want %d", e.Code, exitcode.Timeout)
	}
}

// TestCodeFrom_classifies_raw_k8s_error ensures CodeFrom falls back to
// classification for a raw (unwrapped) Kubernetes error so the exit code still
// propagates through callers that preserve the original cause.
func TestCodeFrom_classifies_raw_k8s_error(t *testing.T) {
	gr := schema.GroupResource{Group: "tekton.dev", Resource: "tasks"}
	raw := k8serrors.NewForbidden(gr, "my-task", fmt.Errorf("nope"))
	if got := exitcode.CodeFrom(raw); got != exitcode.Unauthorized {
		t.Errorf("CodeFrom(raw forbidden) = %d, want %d", got, exitcode.Unauthorized)
	}
}

// TestFromAPIError_preservesK8sErrorDetection ensures the wrapped error still
// matches the k8serrors helpers, so downstream callers that rely on
// IsNotFound/IsForbidden keep working after classification.
func TestFromAPIError_preservesK8sErrorDetection(t *testing.T) {
	gr := schema.GroupResource{Group: "tekton.dev", Resource: "customruns"}

	notFound := exitcode.FromAPIError(k8serrors.NewNotFound(gr, "my-cr"))
	if !k8serrors.IsNotFound(notFound) {
		t.Errorf("IsNotFound(FromAPIError(NotFound)) = false, want true")
	}

	forbidden := exitcode.FromAPIError(k8serrors.NewForbidden(gr, "my-cr", fmt.Errorf("nope")))
	if !k8serrors.IsForbidden(forbidden) {
		t.Errorf("IsForbidden(FromAPIError(Forbidden)) = false, want true")
	}
}
