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

package test

import (
	"fmt"
	"io"

	"github.com/tektoncd/cli/pkg/cmd/hub/app"
	"github.com/tektoncd/cli/pkg/cmd/hub/hub"
)

// API is test URL
const API string = "http://test.hub.cli"

type cli struct {
	hub    hub.Client
	stream app.Stream
}

var _ app.CLI = (*cli)(nil)

func NewCLI(hubType string) app.CLI {
	var h hub.Client
	switch hubType {
	case hub.TektonHubType:
		h = hub.NewTektonHubClient()
	case hub.ArtifactHubType:
		h = hub.NewArtifactHubClient()
	default:
		fmt.Printf("invalid hub type: %s, using default type: %s to continue", hubType, hub.TektonHubType)
		h = hub.NewTektonHubClient()
	}

	if err := h.SetURL(API); err != nil {
		fmt.Printf("Failed validate and set the hub apiURL server URL %v", err)
	}

	return &cli{
		stream: app.Stream{},
		hub:    h,
	}
}

func (c *cli) Stream() app.Stream {
	return c.stream
}

func (c *cli) SetStream(out, err io.Writer) {
	c.stream = app.Stream{Out: out, Err: err}
}

func (c *cli) Hub() hub.Client {
	return c.hub
}

func (c *cli) SetHub(hubType string) error {
	if hubType == hub.TektonHubType {
		c.hub = hub.NewTektonHubClient()
		return nil
	} else if hubType == hub.ArtifactHubType {
		c.hub = hub.NewArtifactHubClient()
		return nil
	}

	return fmt.Errorf("invalid hub type: %s", hubType)
}
