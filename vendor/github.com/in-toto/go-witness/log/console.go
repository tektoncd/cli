// Copyright 2025 The Witness Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"fmt"
)

type ConsoleLogger struct{}

func (ConsoleLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

func (ConsoleLogger) Error(args ...interface{}) {
	fmt.Println(append([]interface{}{"[ERROR]"}, args...)...)
}

func (ConsoleLogger) Warnf(format string, args ...interface{}) {
	fmt.Printf("[WARN] "+format+"\n", args...)
}

func (ConsoleLogger) Warn(args ...interface{}) {
	fmt.Println(append([]interface{}{"[WARN]"}, args...)...)
}

func (ConsoleLogger) Debugf(format string, args ...interface{}) {
	fmt.Printf("[DEBUG] "+format+"\n", args...)
}

func (ConsoleLogger) Debug(args ...interface{}) {
	fmt.Println(append([]interface{}{"[DEBUG]"}, args...)...)
}

func (ConsoleLogger) Infof(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

func (ConsoleLogger) Info(args ...interface{}) {
	fmt.Println(append([]interface{}{"[INFO]"}, args...)...)
}
