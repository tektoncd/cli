// Copyright 2022 The Witness Contributors
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

package timestamp

import (
	"context"
	"fmt"
	"io"
	"time"
)

type FakeTimestamper struct {
	T time.Time
}

func (ft FakeTimestamper) Timestamp(context.Context, io.Reader) ([]byte, error) {
	return []byte(ft.T.Format(time.RFC3339)), nil
}

func (ft FakeTimestamper) Verify(ctx context.Context, ts io.Reader, sig io.Reader) (time.Time, error) {
	b, err := io.ReadAll(ts)
	if err != nil {
		return time.Time{}, err
	}

	if string(b) != ft.T.Format(time.RFC3339) {
		return time.Time{}, fmt.Errorf("mismatched time")
	}

	return ft.T, nil
}
