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

package validate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/bundle"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/exitcode"
)

// Command returns the validate command, which checks Tekton resources locally
// with the same defaulting and validation the Tekton webhook runs, so no
// cluster or credentials are needed.
func Command(_ cli.Params) *cobra.Command {
	var files []string
	eg := `Validate the Tekton resources in a file:

    tkn validate -f pipeline.yaml

Validate several files at once:

    tkn validate -f task.yaml -f pipeline.yaml

Validate resources read from standard input:

    cat pipeline.yaml | tkn validate -f -
`

	c := &cobra.Command{
		Use:     "validate",
		Short:   "Validate Tekton resources in files without a cluster",
		Example: eg,
		Args:    cobra.NoArgs,
		Annotations: map[string]string{
			"commandType": "main",
			"kubernetes":  "false",
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(files) == 0 {
				return exitcode.New(exitcode.InvalidInput, "at least one file is required, use -f")
			}
			return validateFiles(context.Background(), cmd.InOrStdin(), cmd.OutOrStdout(), files)
		},
	}

	c.Flags().StringSliceVarP(&files, "filenames", "f", []string{}, "Files containing Tekton resources in YAML or JSON to validate, use - for standard input")
	return c
}

func validateFiles(ctx context.Context, in io.Reader, out io.Writer, files []string) error {
	var errs []error
	for _, path := range files {
		contents, err := readFile(in, path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := bundle.ValidateTektonObjects(ctx, []string{contents}); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", displayName(path), err))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	fmt.Fprintf(out, "All Tekton resources in %d file(s) are valid\n", len(files))
	return nil
}

func readFile(in io.Reader, path string) (string, error) {
	if path == "-" {
		b, err := io.ReadAll(in)
		if err != nil {
			return "", fmt.Errorf("failed to read standard input: %w", err)
		}
		return string(b), nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", path, err)
	}
	return string(b), nil
}

func displayName(path string) string {
	if path == "-" {
		return "stdin"
	}
	return path
}
