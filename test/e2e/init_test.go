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

package e2e

import (
	"log"
	"os"
	"testing"

	"github.com/tektoncd/cli/test/environment"
)

var TestEnv = &environment.TknRunner{}

func init() {
	var err error
	TestEnv, err = environment.NewTknRunner()
	if err != nil {
		log.Fatalf("Error while creating Env %+v", err)
	}
}

func TestMain(m *testing.M) {
	log.Println("Running main e2e Test suite ")
	v := m.Run()
	os.Exit(v)
}
