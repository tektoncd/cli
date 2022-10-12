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

package log

import (
	"fmt"
	"time"

	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/options"
	"github.com/tektoncd/cli/pkg/pods"
	"github.com/tektoncd/cli/pkg/pods/stream"
)

type Reader struct {
	run             string
	ns              string
	clients         *cli.Clients
	streamer        stream.NewStreamerFunc
	stream          *cli.Stream
	allSteps        bool
	follow          bool
	timestamps      bool
	tasks           []string
	steps           []string
	logType         string
	task            string
	number          int
	activityTimeout time.Duration
	retries         int
}

func NewReader(logType string, opts *options.LogOptions) (*Reader, error) {
	streamer := pods.NewStream
	if opts.Streamer != nil {
		streamer = opts.Streamer
	}

	cs, err := opts.Params.Clients()
	if err != nil {
		return nil, err
	}

	var run string
	switch logType {
	case LogTypePipeline:
		run = opts.PipelineRunName
	case LogTypeTask:
		run = opts.TaskrunName
	}

	at := 10 * time.Second
	if opts.ActivityTimeout != 0 {
		at = opts.ActivityTimeout
	}

	return &Reader{
		run:             run,
		ns:              opts.Params.Namespace(),
		clients:         cs,
		streamer:        streamer,
		stream:          opts.Stream,
		follow:          opts.Follow,
		timestamps:      opts.Timestamps,
		allSteps:        opts.AllSteps,
		tasks:           opts.Tasks,
		steps:           opts.Steps,
		logType:         logType,
		activityTimeout: at,
	}, nil
}

func (r *Reader) Read() (<-chan Log, <-chan error, error) {
	switch r.logType {
	case LogTypePipeline:
		return r.readPipelineLog()
	case LogTypeTask:
		return r.readTaskLog()
	}
	return nil, nil, fmt.Errorf("unknown log type")
}

func (r *Reader) setNumber(number int) {
	r.number = number
}

func (r *Reader) setRetries(retries int) {
	r.retries = retries
}

func (r *Reader) setRun(run string) {
	r.run = run
}

func (r *Reader) setTask(task string) {
	r.task = task
}

func (r *Reader) clone() *Reader {
	c := *r
	return &c
}
