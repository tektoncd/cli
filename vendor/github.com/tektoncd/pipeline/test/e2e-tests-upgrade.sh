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
# and deploy Tekton Pipelines to it for running upgrading tests. There are
# two scenarios we need to cover in this script:

# Scenario 1: install the previous release, upgrade to the current release, and
# validate whether the Tekton pipeline works.

# Scenario 2: install the previous release, create the pipelines and tasks, upgrade
# to the current release, and validate whether the Tekton pipeline works.

source $(dirname $0)/e2e-common.sh
PREVIOUS_PIPELINE_VERSION=v0.5.2

# Script entry point.

initialize $@

header "Setting up environment"

# Handle failures ourselves, so we can dump useful info.
set +o errexit
set +o pipefail

# First, we will verify if Scenario 1 works.
# Install the previous release.
header "Install the previous release of Tekton pipeline $PREVIOUS_PIPELINE_VERSION"
install_pipeline_crd_version $PREVIOUS_PIPELINE_VERSION

# Upgrade to the current release.
header "Upgrade to the current release of Tekton pipeline"
install_pipeline_crd

# Run the integration tests.
failed=0
go_test_e2e -timeout=20m ./test || failed=1

# Run the post-integration tests.
for test in taskrun pipelinerun; do
  header "Running YAML e2e tests for ${test}s"
  if ! run_yaml_tests ${test}; then
    echo "ERROR: one or more YAML tests failed"
    output_yaml_test_results ${test}
    output_pods_logs ${test}
    failed=1
  fi
done

# Remove all the pipeline CRDs, and clean up the environment for next Scenario.
uninstall_pipeline_crd
uninstall_pipeline_crd_version $PREVIOUS_PIPELINE_VERSION

# Next, we will verify if Scenario 2 works.
# Install the previous release.
header "Install the previous release of Tekton pipeline $PREVIOUS_PIPELINE_VERSION"
install_pipeline_crd_version $PREVIOUS_PIPELINE_VERSION

# Create the resources of taskrun and pipelinerun, under the directories example/taskrun
# and example/pipelinerun.
for test in taskrun pipelinerun; do
  header "Applying the resources ${test}s"
  apply_resources ${test}
done

# Upgrade to the current release.
header "Upgrade to the current release of Tekton pipeline"
install_pipeline_crd

# Run the integration tests.
go_test_e2e -timeout=20m ./test || failed=1

# Run the post-integration tests. We do not need to install the resources again, since
# they are installed before the upgrade. We verify if they still work, after going through
# the upgrade.
for test in taskrun pipelinerun; do
  header "Running YAML e2e tests for ${test}s"
  if ! run_tests ${test}; then
    echo "ERROR: one or more YAML tests failed"
    output_yaml_test_results ${test}
    output_pods_logs ${test}
    failed=1
  fi
done

(( failed )) && fail_test

success
