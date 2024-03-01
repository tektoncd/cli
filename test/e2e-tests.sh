#!/usr/bin/env bash

# Copyright 2018 The Tekton Authors
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

# TODO: fix common.sh and  enable set  -euo pipefail
#set -euxo pipefail

source $(dirname $0)/e2e-common.sh

cd $(dirname $(readlink -f $0))/../

E2E_SKIP_CLUSTER_CREATION=${E2E_SKIP_CLUSTER_CREATION:="false"}

ci_run && {
    header "Setting up environment"
    if [ "${E2E_SKIP_CLUSTER_CREATION}" != "true" ]; then
        initialize "$@"
    fi
}

tkn() {
    if [[ -e ./bin/tkn ]];then
        ./bin/tkn $@
    else
        go build github.com/tektoncd/cli/cmd/tkn
        ./tkn "$@"
    fi
}

kubectl get crds|grep tekton\\.dev && fail_test "TektonCD CRDS should not be installed, you should reset them before each runs"

# command before creation of resources
# ensure that the cluster does not have any pipeline resources and that all
# tkn commands would fail
must_fail  "describe pipeline before installing crd" tkn pipeline describe foo
must_fail  "describe pipelinerun before installing crd" tkn pipelinerun describe foo

must_fail  "list pipelinerun for pipeline foo before installing crd" tkn pipelinerun list foo
must_fail  "list pipelines before installing crd" tkn pipeline list
must_fail  "list task before installing crd" tkn task list
must_fail  "list taskrun before installing crd" tkn taskrun list
must_fail  "list taskrun for task foo before installing crd" tkn taskrun list foo

must_fail  "logs of pipelinerun before installing crd" tkn pipelinerun logs foo
must_fail  "logs of taskrun before installing crd" tkn taskrun logs foo

must_fail  "list eventlisteners before installing crd" tkn eventlistener ls
must_fail  "list triggertemplates before installing crd" tkn triggertemplate ls
must_fail  "list triggerbindings before installing crd" tkn triggerbinding ls
must_fail  "list clustertriggerbindings before installing crd" tkn clustertriggerbinding ls

install_pipeline_crd
install_triggers_crd

# listing objects after CRD is created should not fail
kubectl config set-context $(kubectl config current-context) --namespace=default
run_test "show installed versions for triggers and pipelines" tkn version
run_test  "list pipelinerun for pipeline foo" tkn pipelinerun list foo
run_test  "list pipelinerun"  tkn pipelinerun list
run_test  "list pipelines" tkn pipeline list
run_test  "list task" tkn task list
run_test  "list taskrun for task foo" tkn taskrun list foo
run_test  "list taskrun" tkn taskrun list

run_test  "list eventlisteners" tkn eventlistener ls
run_test  "list triggertemplates" tkn triggertemplate ls
run_test  "list triggerbindings" tkn triggerbinding ls
run_test  "list clustertriggerbindings" tkn clustertriggerbinding ls

echo "..............................."

# fetching objects that do not exist must fail
must_fail  "describe pipeline that does not exist" tkn pipeline describe foo
must_fail  "describe pipelinerun that does not exist" tkn pipelinerun describe foo
must_fail  "logs of pipelinerun that does not exist" tkn pipelinerun logs foo
must_fail  "logs of taskrun that does not exist" tkn taskrun logs foo

must_fail  "describe eventlistener that does not exist" tkn eventlistener describe foo
must_fail  "describe triggertemplate that does not exist" tkn triggertemplate describe foo

echo "....................."

# change default namespace to tektoncd
kubectl create namespace tektoncd
kubectl config set-context $(kubectl config current-context) --namespace=tektoncd

# create pipeline, pipelinerun, task, and taskrun
kubectl apply -f ./test/resources/output-pipelinerun.yaml
kubectl apply -f ./test/resources/task-volume.yaml
kubectl apply -f ./test/resources/eventlistener/eventlistener-multi-replica.yaml
echo Waiting for resources to be ready
echo ---------------------------------
wait_until_ready 600 pipelinerun/output-pipeline-run || exit 1
wait_until_ready 600 taskrun/test-template-volume  || exit 1
echo ---------------------------------

# commands after install_pipeline
echo Running tests
echo ---------------------------------
run_test  "list pipelinerun for pipeline output-pipeline" tkn pipelinerun list output-pipeline
run_test  "list pipelinerun" tkn pipelinerun list
run_test  "list pipelines" tkn pipeline list
run_test  "list task" tkn task list
run_test  "list taskrun for pipeline task-volume" tkn taskrun list task-volume
run_test  "list taskrun" tkn taskrun list
run_test  "list eventlistener" tkn eventlistener list

run_test  "describe pipeline" tkn pipeline describe output-pipeline
run_test  "describe pipelinerun" tkn pipelinerun describe output-pipeline-run
run_test  "describe eventlistener" tkn eventlistener describe github-listener-interceptor

run_test  "show logs" tkn pipelinerun logs output-pipeline-run
run_test  "show logs" tkn taskrun logs test-template-volume

# delete pipeline, pipelinerun, task, taskrun, and eventlistener
run_test  "delete pipeline" tkn pipeline delete output-pipeline -f
run_test  "delete pipelinerun" tkn pipelinerun delete output-pipeline-run -f
run_test  "delete task" tkn task delete create-file-verify -f
run_test  "delete taskrun" tkn taskrun delete test-template-volume -f
run_test  "delete eventlistener" tkn eventlistener delete github-listener-interceptor -f

# confirm deletion (TODO: Add task test when added desc or logs to task command)
must_fail  "describe deleted pipeline" tkn pipeline describe output-pipeline
must_fail  "describe deleted pipelinerun" tkn pipelinerun describe output-pipeline-run
must_fail  "show logs deleted taskrun" tkn taskrun logs test-template-volume
must_fail  "describe deleted eventlistener" tkn eventlistener describe github-listener-interceptor

# Hub tests
echo "Hub"
echo "....................."
run_test "hub command exists" tkn hub --help
run_test "hub search exists" tkn hub search --help
run_test "hub get exists" tkn hub get --help
must_fail "hub search cli" tkn hub --api-server=api.nonexistent.server search cli
must_fail "hub get task cli" tkn hub --api-server=api.nonexistent.server get task cli

# Make sure that eveything is cleaned up in the current namespace.
for res in tasks pipelines taskruns pipelineruns; do
  kubectl delete --ignore-not-found=true ${res}.tekton.dev --all
done

# Make sure that eveything is cleaned up in the current namespace.
for res in eventlistener triggertemplate triggerbinding clustertriggerbinding; do
  kubectl delete --ignore-not-found=true ${res}.triggers.tekton.dev --all
done

# Run go e2e tests
header "Running Go e2e tests"
export SYSTEM_NAMESPACE=${SYSTEM_NAMESPACE:-"tekton-pipelines"}
failed=0
if [[ -e ./bin/tkn ]];then
    export TEST_CLIENT_BINARY="${PWD}/bin/tkn"
else
    go build -o tkn github.com/tektoncd/cli/cmd/tkn
    echo "Go Build successfull"
    export TEST_CLIENT_BINARY="${PWD}/tkn"
fi

go_test_e2e ./test/e2e/... || failed=1
(( failed )) && fail_test

success
