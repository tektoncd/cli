// Copyright 2025 The Archivista Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package api

import "net/http"

type RequestOption func(*requestOptions)

type requestOptions struct {
	additionalHeaders http.Header
}

func WithHeaders(h http.Header) RequestOption {
	return func(ro *requestOptions) {
		if h != nil {
			ro.additionalHeaders = h.Clone()
		}
	}
}

func applyRequestOptions(req *http.Request, requestOpts ...RequestOption) *http.Request {
	if req == nil {
		return nil
	}

	opts := &requestOptions{}
	for _, opt := range requestOpts {
		if opt == nil {
			continue
		}

		opt(opts)
	}

	if opts.additionalHeaders != nil {
		req.Header = opts.additionalHeaders
	}

	return req
}
