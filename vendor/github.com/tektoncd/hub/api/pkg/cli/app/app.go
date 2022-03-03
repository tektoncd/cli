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

package app

import (
	"io"

	"github.com/tektoncd/hub/api/pkg/cli/hub"
)

type Stream struct {
	Out io.Writer
	Err io.Writer
}

type CLI interface {
	Hub() hub.Client
	Stream() Stream
	SetStream(out, err io.Writer)
}

type cli struct {
	hub    hub.Client
	stream Stream
}

var _ CLI = (*cli)(nil)

func New() *cli {
	return &cli{hub: hub.NewClient()}
}

func (c *cli) Stream() Stream {
	return c.stream
}

func (c *cli) SetStream(out, err io.Writer) {
	c.stream = Stream{Out: out, Err: err}
}

func (c *cli) Hub() hub.Client {
	return c.hub
}
