/*
Copyright 2025 The Tekton Authors
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

package metrics

import "context"

// Metric is a string name of a well-known metric
type Metric string

// Common metric names
const (
	SignedMessagesCount Metric = "sgcount"
	SignsStoredCount    Metric = "stcount"
	PayloadUploadeCount Metric = "plcount"
	MarkedAsSignedCount Metric = "mrcount"
)

// MetricErrorType is a string name of a well-known error type. Any error metrics recorded should be faceted by MetricErrorType
type MetricErrorType string

// Common error types
const (
	PayloadCreationError MetricErrorType = "payload_creation"
	MarshalPayloadError  MetricErrorType = "marshal_payload"
	SigningError         MetricErrorType = "signing"
	StorageError         MetricErrorType = "storage"
	TlogError            MetricErrorType = "tlog"
)

// Recorder is an interface for any object which may record metrics
type Recorder interface {
	RecordCountMetrics(ctx context.Context, MetricType Metric)
	RecordErrorMetric(ctx context.Context, errType MetricErrorType)
}
