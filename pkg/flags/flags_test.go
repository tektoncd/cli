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

package flags

import (
	"testing"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"gotest.tools/v3/assert"
)

func TestFlags_colouring(t *testing.T) {
	// When running it on CI, our test don't have a tty so this gets disabled
	// automatically, not really sure how can we workaround that :(
	// cmd := &cobra.Command{}
	// cmd.SetArgs([]string{"--nocolour"})
	// _ = InitParams(&cli.TektonParams{}, cmd)
	// assert.False(t, color.NoColor)

	_ = InitParams(&cli.TektonParams{}, &cobra.Command{})
	assert.Assert(t, color.NoColor == true)

}
