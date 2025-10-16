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

func DownloadReadCloser(ctx context.Context, baseURL string, gitoid string, requestOptions ...RequestOption) (io.ReadCloser, error) {
	return DownloadReadCloserWithHTTPClient(ctx, &http.Client{}, baseURL, gitoid, requestOptions...)
}

func DownloadReadCloserWithHTTPClient(ctx context.Context, client *http.Client, baseURL string, gitoid string, requestOptions ...RequestOption) (io.ReadCloser, error) {
	downloadURL, err := url.JoinPath(baseURL, "download", gitoid)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}

	req = applyRequestOptions(req, requestOptions...)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		// NOTE: attempt to read body on error and
		// only close if an error occurs
		defer resp.Body.Close()
		errMsg, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(string(errMsg))
	}
	return resp.Body, nil
}

func Download(ctx context.Context, baseURL string, gitoid string, requestOptions ...RequestOption) (dsse.Envelope, error) {
	buf := &bytes.Buffer{}
	if err := DownloadWithWriter(ctx, baseURL, gitoid, buf, requestOptions...); err != nil {
		return dsse.Envelope{}, err
	}

	env := dsse.Envelope{}
	dec := json.NewDecoder(buf)
	if err := dec.Decode(&env); err != nil {
		return env, err
	}

	return env, nil
}

func DownloadWithWriter(ctx context.Context, baseURL string, gitoid string, dst io.Writer, requestOptions ...RequestOption) error {
	return DownloadWithWriterWithHTTPClient(ctx, &http.Client{}, baseURL, gitoid, dst, requestOptions...)
}

func DownloadWithWriterWithHTTPClient(ctx context.Context, client *http.Client, baseURL string, gitoid string, dst io.Writer, requestOptions ...RequestOption) error {
	downloadUrl, err := url.JoinPath(baseURL, "download", gitoid)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadUrl, nil)
	if err != nil {
		return err
	}

	req = applyRequestOptions(req, requestOptions...)
	req.Header.Set("Content-Type", "application/json")
	hc := &http.Client{}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errMsg, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return errors.New(string(errMsg))
	}

	_, err = io.Copy(dst, resp.Body)
	return err
}
