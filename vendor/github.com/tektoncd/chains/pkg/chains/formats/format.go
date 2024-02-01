/*
Copyright 2020 The Tekton Authors
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

package formats

import (
	"context"
	"fmt"

	"github.com/tektoncd/chains/pkg/config"
)

// Payloader is an interface to generate a chains Payload from a TaskRun
type Payloader interface {
	CreatePayload(ctx context.Context, obj interface{}) (interface{}, error)
	Type() config.PayloadType
	Wrap() bool
}

const (
	PayloadTypeTekton        config.PayloadType = "tekton"
	PayloadTypeSimpleSigning config.PayloadType = "simplesigning"
	PayloadTypeInTotoIte6    config.PayloadType = "in-toto"
	PayloadTypeSlsav1        config.PayloadType = "slsa/v1"
	PayloadTypeSlsav2alpha1  config.PayloadType = "slsa/v2alpha1"
	PayloadTypeSlsav2alpha2  config.PayloadType = "slsa/v2alpha2"
	PayloadTypeSlsav2alpha3  config.PayloadType = "slsa/v2alpha3"
)

var (
	IntotoAttestationSet = map[config.PayloadType]struct{}{
		PayloadTypeInTotoIte6:   {},
		PayloadTypeSlsav1:       {},
		PayloadTypeSlsav2alpha1: {},
		PayloadTypeSlsav2alpha2: {},
		PayloadTypeSlsav2alpha3: {},
	}
	payloaderMap = map[config.PayloadType]PayloaderInit{}
)

// PayloaderInit initializes a new Payloader instance for the given config.
type PayloaderInit func(config.Config) (Payloader, error)

// RegisterPayloader registers the PayloaderInit func for the given type.
// This is suitable to be calling during init() to register Payloader types.
func RegisterPayloader(key config.PayloadType, init PayloaderInit) {
	payloaderMap[key] = init
}

// GetPayloader returns a new Payloader of the given type.
// If no Payloader is registered for the type, an error is returned.
func GetPayloader(key config.PayloadType, cfg config.Config) (Payloader, error) {
	fn, ok := payloaderMap[key]
	if !ok {
		return nil, fmt.Errorf("payloader %q not found", key)
	}
	return fn(cfg)
}
