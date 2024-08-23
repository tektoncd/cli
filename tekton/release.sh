#!/usr/bin/env bash
RELEASE_VERSION="${1}"
COMMITS="${2}"

UPSTREAM_REMOTE=upstream
DEFAULT_BRANCH="main"
TARGET_NAMESPACE="release"
SECRET_NAME=bot-token-github
PUSH_REMOTE="${PUSH_REMOTE:-${UPSTREAM_REMOTE}}" # export PUSH_REMOTE to your own for testing

CATALOG_TASKS="lint build test"

BINARIES="kubectl jq tkn git"

GOLANGCI_VERSION="$(cat tools/go.mod | grep golangci-lint | awk '{ print $3 }')"
GO_VERSION="$(cat go.mod | grep "go" | awk 'NR==1{ print $2 }')"

set -e

for b in ${BINARIES};do
    type -p ${b} >/dev/null || { echo "'${b}' need to be avail"; exit 1 ;}
done


kubectl version 2>/dev/null >/dev/null || {
    echo "you need to have access to a kubernetes cluster"
    exit 1
}

kubectl get pipeline 2>/dev/null >/dev/null || {
    echo "you need to have tekton install onto the cluster"
    exit 1
}

[[ -z ${RELEASE_VERSION} ]] && {
    read -e -p "Enter a target release (i.e: v0.1.2): " RELEASE_VERSION
    [[ -z ${RELEASE_VERSION} ]] && { echo "no target release"; exit 1 ;}
}

[[ ${RELEASE_VERSION} =~ v[0-9]+\.[0-9]*\.[0-9]+ ]] || { echo "invalid version provided, need to match v\d+\.\d+\.\d+"; exit 1 ;}
git fetch -a --tags ${UPSTREAM_REMOTE} >/dev/null
lasttag=$(git describe --tags `git rev-list --tags --max-count=1`)
echo ${lasttag}|sed 's/\.[0-9]*$//'|grep -q ${RELEASE_VERSION%.*} && {
    echo "Minor version of ${RELEASE_VERSION%.*} detected, previous ${lasttag}"; minor_version=true ;
} || {
    echo "Major version for ${RELEASE_VERSION%.*} detected, previous ${lasttag}";
}

cd ${GOPATH}/src/github.com/tektoncd/cli

[[ -n $(git status --porcelain 2>&1) ]] && {
    echo "We have detected some changes in your repo"
    echo "Stash them before executing this script"
    exit 1
}

git checkout ${DEFAULT_BRANCH}
git reset --hard ${UPSTREAM_REMOTE}/${DEFAULT_BRANCH}
git checkout -B release-${RELEASE_VERSION}  ${DEFAULT_BRANCH} >/dev/null

if [[ -n ${minor_version} ]];then
    git reset --hard ${lasttag} >/dev/null

    if [[ -z ${COMMITS} ]];then
        echo "Showing commit between last minor tag '${lasttag} to '${DEFAULT_BRANCH}'"
        echo
        git log --reverse --no-merges --pretty=format:"%C(bold cyan)%h%Creset | %cd | %s | %ae" ${DEFAULT_BRANCH} --since "$(git log --pretty=format:%cd -1 ${lasttag})"
        echo
        read -e -p "Pick a list of ordered commits to cherry-pick space separated (* mean all of them): " COMMITS
    fi
    [[ -z ${COMMITS} ]] && { echo "no commits picked"; exit 1;}
    if [[ ${COMMITS} == "*" ]];then
        COMMITS=$(git log --reverse --no-merges --pretty=format:"%h" ${DEFAULT_BRANCH} \
                      --since "$(git log --pretty=format:%cd -1 ${lasttag})")
    fi
    for commit in ${COMMITS};do
        git branch --contains ${commit} >/dev/null || { echo "Invalid commit ${commit}" ; exit 1;}
        echo "Cherry-picking commit: ${commit}"
        git cherry-pick ${commit} >/dev/null
    done
else
    echo "Major release ${RELEASE_VERSION%.*} detected: picking up ${UPSTREAM_REMOTE}/${DEFAULT_BRANCH}"
    git reset --hard ${UPSTREAM_REMOTE}/${DEFAULT_BRANCH}
fi

# HACK:! this is temporary to disable the upload to homebrew when we do our testing
# need to find a way how to do this automatically (push to upstream/revert and
# then being able to cherry-pick if $PUSH_REMOTE != 'upstream/origin'?)
[[ ${PUSH_REMOTE} != origin* && ${PUSH_REMOTE} != upstream* ]] && {
    echo "Testing mode detected Cherry-picking commit: 052b0b4ce989fe9aee01027e67e61538b48e1179"
    git cherry-pick 052b0b4ce989fe9aee01027e67e61538b48e1179 >/dev/null
}

COMMITS=$(git log --reverse --no-merges \
              --pretty=format:'%H' ${DEFAULT_BRANCH} \
              --since "$(git log --pretty=format:%cd -1 ${lasttag})")

changelog=""
for c in ${COMMITS};do
    pr=$(curl -s "https://api.github.com/search/issues?q=${c}" |jq -r '.items[0].html_url')
    pr=$(echo ${pr}|sed 's,https://github.com/tektoncd/cli/pull/,tektoncd/cli#,')
    changelog+="${pr} | $(git log -1 --date=format:'%Y/%m/%d-%H:%M' --pretty=format:'[%an] %s | %cd' ${c})
"
done

# Add our VERSION so Makefile will pick it up when compiling
echo ${RELEASE_VERSION#v} > VERSION
git add VERSION
git commit -sm "New version ${RELEASE_VERSION}" -m "${changelog}" VERSION
git tag --sign -m \
    "New version ${RELEASE_VERSION}" \
    -m "${changelog}" --force ${RELEASE_VERSION}

git push --force ${PUSH_REMOTE} ${RELEASE_VERSION}
git push --force ${PUSH_REMOTE} release-${RELEASE_VERSION}

kubectl create namespace ${TARGET_NAMESPACE} 2>/dev/null || true

for task in ${CATALOG_TASKS};do
  if [ ${task} == "lint" ]; then
    tkn -n ${TARGET_NAMESPACE} hub install task golangci-lint
  else
    tkn -n ${TARGET_NAMESPACE} hub install task golang-${task}
  fi
done

tkn -n ${TARGET_NAMESPACE} hub install task git-clone
tkn -n ${TARGET_NAMESPACE} hub install task goreleaser

kubectl -n ${TARGET_NAMESPACE} apply -f ./tekton/get-version.yaml
kubectl -n ${TARGET_NAMESPACE} apply -f ./tekton/publish.yaml

if ! kubectl -n ${TARGET_NAMESPACE} get secret ${SECRET_NAME} -o name >/dev/null 2>/dev/null;then
    github_token=$(git config --get github.oauth-token || true)

    [[ -z ${github_token} ]] && {
        read -e -p "Enter your Github Token: " github_token
        [[ -z ${github_token} ]] && { echo "no token provided"; exit 1;}
    }

    kubectl -n ${TARGET_NAMESPACE} create secret generic ${SECRET_NAME} --from-literal=bot-token=${github_token}
fi

kubectl -n ${TARGET_NAMESPACE} apply -f ./tekton/release-pipeline.yml

sleep 2

URL=$(git remote get-url "${PUSH_REMOTE}" | sed 's,git@github.com:,https://github.com/,')
# Start the pipeline
tkn -n ${TARGET_NAMESPACE} pipeline start cli-release-pipeline \
  -p revision="${RELEASE_VERSION}" \
  -p url="${URL}" \
  -p golangci-lint-version="${GOLANGCI_VERSION}" \
  -p go-version="${GO_VERSION}" \
  -w name=shared-workspace,volumeClaimTemplateFile=./tekton/volume-claim-template.yml

tkn -n ${TARGET_NAMESPACE} pipeline logs cli-release-pipeline -f --last
