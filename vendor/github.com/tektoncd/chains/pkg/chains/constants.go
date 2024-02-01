/*
Copyright 2024 The Tekton Authors
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

package chains

const (
	SignedMessagesCount     = "sgcount"
	SignsStoredCount        = "stcount"
	PayloadUploadeCount     = "plcount"
	MarkedAsSignedCount     = "mrcount"
	PipelineRunSignedName   = "pipelinerun_sign_created_total"
	PipelineRunSignedDesc   = "Total number of signed messages for pipelineruns"
	PipelineRunUploadedName = "pipelinerun_payload_uploaded_total"
	PipelineRunUploadedDesc = "Total number of uploaded payloads for pipelineruns"
	PipelineRunStoredName   = "pipelinerun_payload_stored_total"
	PipelineRunStoredDesc   = "Total number of stored payloads for pipelineruns"
	PipelineRunMarkedName   = "pipelinerun_marked_signed_total"
	PipelineRunMarkedDesc   = "Total number of objects marked as signed for pipelineruns"
	TaskRunSignedName       = "taskrun_sign_created_total"
	TaskRunSignedDesc       = "Total number of signed messages for taskruns"
	TaskRunUploadedName     = "taskrun_payload_uploaded_total"
	TaskRunUploadedDesc     = "Total number of uploaded payloads for taskruns"
	TaskRunStoredName       = "taskrun_payload_stored_total"
	TaskRunStoredDesc       = "Total number of stored payloads for taskruns"
	TaskRunMarkedName       = "taskrun_marked_signed_total"
	TaskRunMarkedDesc       = "Total number of objects marked as signed for taskruns"
)
