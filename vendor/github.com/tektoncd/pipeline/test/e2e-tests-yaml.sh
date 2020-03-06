#!/usr/bin/env bash

# Copyright 2019 The Tekton Authors
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

# This script calls out to scripts in tektoncd/plumbing to setup a cluster
# and deploy Tekton Pipelines to it for running integration tests.

source $(git rev-parse --show-toplevel)/test/e2e-common.sh

# Script entry point.

initialize $@

header "Setting up environment"

# Handle failures ourselves, so we can dump useful info.
set +o errexit
set +o pipefail

install_pipeline_crd

# Run the tests
failed=0
for version in v1alpha1 v1beta1; do
  for test in taskrun pipelinerun; do
    header "Running YAML e2e tests for ${version} ${test}s"
    if ! run_yaml_tests ${version} ${test}; then
      echo "ERROR: one or more YAML tests failed"
      output_yaml_test_results ${test}
      output_pods_logs ${test}
      failed=1
    fi
  done
  # Clean resources
  delete_pipeline_resources
  for res in services pods configmaps secrets serviceaccounts persistentvolumeclaims; do
    kubectl delete --ignore-not-found=true ${res} --all
  done
done

(( failed )) && fail_test

success
