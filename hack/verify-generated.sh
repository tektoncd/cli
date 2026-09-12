#!/usr/bin/env bash

# Copyright 2026 The Tekton Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Verifies that docs and golden files are up to date.
# Run `make generated` and commit the result if this fails.

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "${REPO_ROOT}"

echo "Regenerating docs..."
make docs

echo "Checking for uncommitted changes in docs..."
git_status=$(git status --porcelain -- docs/)
if [[ -n "$git_status" ]]; then
  echo "$git_status"
  echo ""
  echo "ERROR: docs are out of date."
  echo "Run 'make docs' and commit the result."
  exit 1
fi

echo "All generated docs are up to date."
