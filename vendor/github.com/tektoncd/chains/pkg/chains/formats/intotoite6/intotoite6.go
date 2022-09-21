/*
Copyright 2021 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package intotoite6

import (
	"fmt"

	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/formats/intotoite6/pipelinerun"
	"github.com/tektoncd/chains/pkg/chains/formats/intotoite6/taskrun"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"go.uber.org/zap"
)

type InTotoIte6 struct {
	builderID string
	logger    *zap.SugaredLogger
}

func NewFormatter(cfg config.Config, logger *zap.SugaredLogger) (formats.Payloader, error) {
	return &InTotoIte6{
		builderID: cfg.Builder.ID,
		logger:    logger,
	}, nil
}

func (i *InTotoIte6) Wrap() bool {
	return true
}

func (i *InTotoIte6) CreatePayload(obj interface{}) (interface{}, error) {
	switch v := obj.(type) {
	case *objects.TaskRunObject:
		return taskrun.GenerateAttestation(i.builderID, v, i.logger)
	case *objects.PipelineRunObject:
		return pipelinerun.GenerateAttestation(i.builderID, v, i.logger)
	default:
		return nil, fmt.Errorf("intoto does not support type: %s", v)
	}
}

func (i *InTotoIte6) Type() formats.PayloadType {
	return formats.PayloadTypeInTotoIte6
}
