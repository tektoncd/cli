// Copyright © 2021 The Tekton Authors.
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
package bundle

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/bundle"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/params"
)

const (
	sourceDateEpochEnv = "SOURCE_DATE_EPOCH"
	defaultTimestamp   = 0
)

type pushOptions struct {
	cliparams          cli.Params
	stream             *cli.Stream
	ref                name.Reference
	bundleContents     []string
	bundleContentPaths []string
	remoteOptions      bundle.RemoteOptions
	annotationParams   []string
	annotations        map[string]string
	labelParams        []string
	labels             map[string]string
	ctimeParam         string
	ctime              time.Time
}

func pushCommand(_ cli.Params) *cobra.Command {
	opts := &pushOptions{}

	longHelp := `Publish a new Tekton Bundle to a registry by passing in a set of Tekton objects via files, arguments or standard in:

	tkn bundle push docker.io/myorg/mybundle:latest "apiVersion: tekton.dev/v1beta1 kind: Pipeline..."
	tkn bundle push docker.io/myorg/mybundle:1.0 -f path/to/my/file.json
	cat path/to/my/unified_yaml_file.yaml | tkn bundle push myprivateregistry.com/myorg/mybundle -f -

Authentication:
	There are three ways to authenticate against your registry.
	1. By default, your docker.config in your home directory and podman's auth.json are used.
	2. Additionally, you can supply a Bearer Token via --remote-bearer
	3. Additionally, you can use Basic auth via --remote-username and --remote-password

Input:
	Valid input in any form is valid Tekton YAML or JSON with a fully-specified "apiVersion" and "kind". To pass multiple objects in a single input, use "---" separators in YAML or a top-level "[]" in JSON.

Created time:
	The default created time of the OCI Image Configuration layer is set to 1970-01-01T00:00:00Z. Changing it can be done by either providing it via --ctime parameter or setting the SOURCE_DATE_EPOCH environment variable.
`

	c := &cobra.Command{
		Use:   "push",
		Short: "Create or replace a Tekton bundle",
		Long:  longHelp,
		Annotations: map[string]string{
			"commandType": "main",
			"kubernetes":  "false",
		},
		Args: cobra.ExactArgs(1),
		PreRunE: func(_ *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errInvalidRef
			}

			if _, err := name.ParseReference(args[0], name.StrictValidation, name.Insecure); err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.stream = &cli.Stream{
				In:  cmd.InOrStdin(),
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			return opts.Run(args)
		},
	}
	c.Flags().StringSliceVarP(&opts.bundleContentPaths, "filenames", "f", []string{}, "List of fully-qualified file paths containing YAML or JSON defined Tekton objects to include in this bundle")
	c.Flags().StringSliceVarP(&opts.annotationParams, "annotate", "", []string{}, "OCI Manifest annotation in the form of key=value to be added to the OCI image. Can be provided multiple times to add multiple annotations.")
	c.Flags().StringVar(&opts.ctimeParam, "ctime", "", "YYYY-MM-DD, YYYY-MM-DDTHH:MM:SS or RFC3339 formatted created time to set, defaults to current time. In non RFC3339 syntax dates are in UTC timezone.")
	c.Flags().StringSliceVarP(&opts.labelParams, "label", "", []string{}, "OCI Config labels in the form of key=value to be added to the OCI image. Can be provided multiple times to add multiple labels.")
	bundle.AddRemoteFlags(c.Flags(), &opts.remoteOptions)

	return c
}

// Reads the positional arguments and the `-f` flag to fill in the `bunldeContents` parameter with all of the raw Tekton
// contents.
func (p *pushOptions) parseArgsAndFlags(args []string) (err error) {
	p.ref, _ = name.ParseReference(args[0], name.StrictValidation, name.Insecure)

	// If there are file paths specified, then read them and include their contents.
	for _, path := range p.bundleContentPaths {
		if path == "-" {
			// If this flag's value is '-', assume the user has piped input into stdin.
			stdinContents, err := io.ReadAll(p.stream.In)
			if err != nil {
				return fmt.Errorf("failed to read bundle contents from stdin: %w", err)
			}
			if len(stdinContents) == 0 {
				return errors.New("failed to read bundle contents from stdin: empty input")
			}
			p.bundleContents = append(p.bundleContents, string(stdinContents))
			continue
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to find and read file %s: %w", path, err)
		}
		p.bundleContents = append(p.bundleContents, string(contents))
	}

	if p.annotations, err = params.ParseParams(p.annotationParams); err != nil {
		return err
	}

	if p.labels, err = params.ParseParams(p.labelParams); err != nil {
		return err
	}

	if p.ctime, err = determineCTime(p.ctimeParam); err != nil {
		return err
	}

	return nil
}

// Run performs the principal logic of reading and parsing the input, creating the bundle, and publishing it.
func (p *pushOptions) Run(args []string) error {
	if err := p.parseArgsAndFlags(args); err != nil {
		return err
	}

	img, err := bundle.BuildTektonBundle(p.bundleContents, p.annotations, p.labels, p.ctime, p.stream.Out)
	if err != nil {
		return err
	}

	outputDigest, err := bundle.Write(img, p.ref, p.remoteOptions.ToOptions()...)
	if err != nil {
		return err
	}
	fmt.Fprintf(p.stream.Out, "\nPushed Tekton Bundle to %s\n", outputDigest)
	return err
}

func determineCTime(t string) (parsed time.Time, err error) {
	// if given the parameter don't lookup the SOURCE_DATE_EPOCH env var
	if t == "" {
		if sourceDateEpoch, found := os.LookupEnv(sourceDateEpochEnv); found && sourceDateEpoch != "" {
			timestamp, err := strconv.ParseInt(sourceDateEpoch, 10, 64)
			if err != nil {
				// rather than ignore, report that SOURCE_DATE_EPOCH cannot be
				// parsed, given that it is set seems like the best option
				return time.Time{}, err
			}
			return time.Unix(timestamp, 0), nil
		}
	}

	if t == "" {
		return time.Unix(defaultTimestamp, 0), nil
	}

	parsed, err = time.Parse(time.DateOnly, t)

	if err != nil {
		parsed, err = time.Parse("2006-01-02T15:04:05", t)
	}

	if err != nil {
		parsed, err = time.Parse(time.RFC3339, t)
	}

	if err != nil {
		return parsed, fmt.Errorf("unable to parse provided time %q: %w", t, err)
	}

	return parsed, nil
}
