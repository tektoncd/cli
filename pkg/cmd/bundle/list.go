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
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/bundle"
	"github.com/tektoncd/cli/pkg/cli"
	"k8s.io/apimachinery/pkg/runtime"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
)

// TODO: Find a more generic way to do this
var (
	allowedKinds = []string{"task", "pipeline"}
)

func normalizeKind(kind string) string {
	return strings.TrimSuffix(strings.ToLower(kind), "s")
}

type listOptions struct {
	cliparams     cli.Params
	stream        *cli.Stream
	ref           name.Reference
	remoteOptions bundle.RemoteOptions
	cacheOptions  bundle.CacheOptions
}

func listCommand(p cli.Params) *cobra.Command {
	opts := &listOptions{
		cliparams: p,
	}
	f := cliopts.NewPrintFlags("")

	longHelp := `List the contents of a Tekton Bundle from a registry. You can further narrow down the results by
optionally specifying the kind, and then the name:

	tkn bundle list docker.io/myorg/mybundle:latest // fetches all objects
	tkn bundle list docker.io/myorg/mybundle:1.0 task // fetches all Tekton tasks
	tkn bundle list docker.io/myorg/mybundle:1.0 task foo // fetches the Tekton task "foo"

As with other "list" commands, you can specify the desired output format using the "-o" flag. You may specify the kind
in its "Kind" form (eg Task), its "Resource" form (eg tasks), or in the form specified by the Tekton Bundle contract (
eg task).

Authentication:
	There are three ways to authenticate against your registry.
	1. By default, your docker.config in your home directory and podman's auth.json are used.
	2. Additionally, you can supply a Bearer Token via --remote-bearer
	3. Additionally, you can use Basic auth via --remote-username and --remote-password

Caching:
    By default, bundles will be cached in ~/.tekton/bundles. If you would like to use a different location, set
"--cache-dir" and if you would like to skip the cache altogether, set "--no-cache".
`

	c := &cobra.Command{
		Use:   "list",
		Short: "List and print a Tekton bundle's contents",
		Long:  longHelp,
		Annotations: map[string]string{
			"commandType": "main",
			"kubernetes":  "false",
		},
		Args: cobra.RangeArgs(1, 3),
		PreRunE: func(_ *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errInvalidRef
			}

			ref, err := name.ParseReference(args[0], name.StrictValidation, name.Insecure)
			if err != nil {
				return err
			}
			opts.ref = ref

			if len(args) > 1 {
				allowedKind := false
				for _, op := range allowedKinds {
					if normalizeKind(args[1]) == op {
						allowedKind = true
					}
				}

				if !allowedKind {
					return fmt.Errorf("second argument %s is not a valid kind: %q", args[1], allowedKinds)
				}
			}

			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.stream = &cli.Stream{
				In:  cmd.InOrStdin(),
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			p, err := f.ToPrinter()
			if err != nil {
				return err
			}

			return opts.Run(args, func(_, _, _ string, element runtime.Object, _ []byte) {
				_ = p.PrintObj(element, opts.stream.Out)
			})
		},
	}

	f.AddFlags(c)
	bundle.AddRemoteFlags(c.Flags(), &opts.remoteOptions)
	bundle.AddCacheFlags(c.Flags(), &opts.cacheOptions)

	return c
}

// Run performs the principal logic of reading and parsing the input, creating the bundle, and publishing it.
func (l *listOptions) Run(args []string, formatter bundle.ObjectVisitor) error {
	img, err := bundle.Read(l.ref, &l.cacheOptions, l.remoteOptions.ToOptions()...)
	if err != nil {
		return err
	}

	switch len(args) {
	case 1:
		if err := bundle.List(img, formatter); err != nil {
			return err
		}
	case 2:
		if err := bundle.ListKind(img, normalizeKind(args[1]), formatter); err != nil {
			return err
		}
	case 3:
		if err := bundle.Get(img, normalizeKind(args[1]), args[2], formatter); err != nil {
			return err
		}
	}
	return nil
}
