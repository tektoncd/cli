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

package formatted

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTimeout(t *testing.T) {
	tcs := []struct {
		duration *metav1.Duration
		expected string
	}{
		{
			&metav1.Duration{
				Duration: 5 * time.Minute,
			},
			"5 minutes",
		},
		{
			nil,
			"---",
		},
		{
			&metav1.Duration{
				Duration: 1*time.Hour + 30*time.Minute,
			},
			"1 hour 30 minutes",
		},
	}

	for _, tc := range tcs {
		str := Timeout(tc.duration)
		assert.Equal(t, str, tc.expected)
	}
}

func TestDuration(t *testing.T) {
	now := time.Now()
	startTime := metav1.NewTime(now)
	plusFifteen := metav1.NewTime(now.Add(time.Minute * 15))
	plusOneHour := metav1.NewTime(now.Add(time.Hour * 1))
	plusOneHourAndHalf := metav1.NewTime(now.Add(time.Hour*1 + time.Minute*30))
	plusOneDayAndHalf := metav1.NewTime(now.Add(time.Hour*24 + time.Minute*35))

	tcs := []struct {
		startTime    *metav1.Time
		completeTime *metav1.Time
		expected     string
	}{
		{
			nil,
			nil,
			"---",
		},
		{
			&startTime,
			&plusFifteen,
			"15m0s",
		},
		{
			&startTime,
			&plusOneHour,
			"1h0m0s",
		},
		{
			&startTime,
			&plusOneHourAndHalf,
			"1h30m0s",
		},
		{
			&startTime,
			&plusOneDayAndHalf,
			"24h35m0s",
		},
	}

	for _, tc := range tcs {
		str := Duration(tc.startTime, tc.completeTime)
		assert.Equal(t, str, tc.expected)
	}
}
