// Copyright 2023-2024 The Archivista Contributors
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

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/in-toto/go-witness/dsse"
)

type UploadResponse struct {
	Gitoid string `json:"gitoid"`
}

// Deprecated: Use UploadResponse instead. It will be removed in version >= v0.6.0
type StoreResponse = UploadResponse

// Deprecated: Use Store instead. It will be removed in version >= v0.6.0
func Upload(ctx context.Context, baseURL string, envelope dsse.Envelope, requestOptions ...RequestOption) (UploadResponse, error) {
	return Store(ctx, baseURL, envelope, requestOptions...)
}

func Store(ctx context.Context, baseURL string, envelope dsse.Envelope, requestOptions ...RequestOption) (StoreResponse, error) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	if err := enc.Encode(envelope); err != nil {
		return StoreResponse{}, err
	}

	return StoreWithReader(ctx, baseURL, buf, requestOptions...)
}

func StoreWithReader(ctx context.Context, baseURL string, r io.Reader, requestOptions ...RequestOption) (StoreResponse, error) {
	return StoreWithReaderWithHTTPClient(ctx, &http.Client{}, baseURL, r, requestOptions...)
}

func StoreWithReaderWithHTTPClient(ctx context.Context, client *http.Client, baseURL string, r io.Reader, requestOptions ...RequestOption) (StoreResponse, error) {
	uploadPath, err := url.JoinPath(baseURL, "upload")
	if err != nil {
		return UploadResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", uploadPath, r)
	if err != nil {
		return UploadResponse{}, err
	}

	req = applyRequestOptions(req, requestOptions...)
	req.Header.Set("Content-Type", "application/json")
	hc := &http.Client{}
	resp, err := hc.Do(req)
	if err != nil {
		return UploadResponse{}, err
	}

	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return UploadResponse{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return UploadResponse{}, errors.New(string(bodyBytes))
	}

	uploadResp := UploadResponse{}
	if err := json.Unmarshal(bodyBytes, &uploadResp); err != nil {
		return UploadResponse{}, err
	}

	return uploadResp, nil
}
