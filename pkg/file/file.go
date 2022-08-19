// Copyright © 2019 The Tekton Authors.
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

package file

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

type TypeValidator func(target string) bool

func IsYamlFile() TypeValidator {
	return func(target string) bool {
		if strings.HasSuffix(target, ".yaml") || strings.HasSuffix(target, ".yml") {
			return true
		}
		return false
	}
}

func LoadFileContent(httpClient http.Client, target string, validate TypeValidator, errorMsg error) ([]byte, error) {
	if !validate(target) {
		return nil, errorMsg
	}

	var content []byte
	var err error
	if strings.HasPrefix(target, "http") {
		content, err = getRemoteContent(httpClient, target)
	} else {
		content, err = os.ReadFile(target)
	}

	if err != nil {
		return nil, err
	}

	return content, nil
}

func getRemoteContent(httpClient http.Client, url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("url specified returned a 404: not found")
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	content := buf.Bytes()
	return content, nil
}
