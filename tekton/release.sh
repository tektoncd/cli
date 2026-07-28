#!/usr/bin/env bash
RELEASE_VERSION="${1}"

UPSTREAM_REMOTE=upstream
DEFAULT_BRANCH="main"
TARGET_NAMESPACE="release"
SECRET_NAME=bot-token-github
PUSH_REMOTE="${PUSH_REMOTE:-${UPSTREAM_REMOTE}}" # export PUSH_REMOTE to your own for testing

CATALOG_TASKS="lint build test"

BINARIES="kubectl jq tkn git"

GOLANGCI_VERSION="$(grep -A2 'golangci-lint' .github/workflows/ci.yaml | grep 'version:' | awk '{ print $2 }')"
[[ -z ${GOLANGCI_VERSION} ]] && {
    echo "WARNING: unable to detect golangci-lint version from .github/workflows/ci.yaml, using 'latest'"
    GOLANGCI_VERSION="latest"
}
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

MINOR_BRANCH="release-${RELEASE_VERSION%.*}.x"

git fetch -a --tags ${UPSTREAM_REMOTE} >/dev/null

git ls-remote --exit-code ${UPSTREAM_REMOTE} refs/heads/${MINOR_BRANCH} >/dev/null 2>&1 && {
    patch_release=true
} || true

if [[ -n ${patch_release} ]];then
    RELEASE_BRANCH="release-${RELEASE_VERSION}"
    version_prefix="${RELEASE_VERSION%.*}"
    prev_tag=$(git tag --sort=-v:refname -l "${version_prefix}.*" | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | grep -v "^${RELEASE_VERSION}$" | head -1)
    [[ -z ${prev_tag} ]] && { echo "no previous patch release tag found for ${version_prefix}"; exit 1; }
    echo "Patch release detected: creating ${RELEASE_BRANCH} from ${MINOR_BRANCH} on ${UPSTREAM_REMOTE}, previous tag ${prev_tag}"
else
    RELEASE_BRANCH="${MINOR_BRANCH}"
    prev_tag=$(git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.0$' | grep -v "^${RELEASE_VERSION}$" | head -1)
    [[ -z ${prev_tag} ]] && { echo "no previous release tag found for ${RELEASE_VERSION}"; exit 1; }
    echo "New minor release detected: creating ${RELEASE_BRANCH} from ${DEFAULT_BRANCH}, previous tag ${prev_tag}"
fi

cd ${GOPATH}/src/github.com/tektoncd/cli

[[ -n $(git status --porcelain 2>&1) ]] && {
    echo "We have detected some changes in your repo"
    echo "Stash them before executing this script"
    exit 1
}

if [[ -n ${patch_release} ]];then
    echo "Creating ${RELEASE_BRANCH} from ${UPSTREAM_REMOTE}/${MINOR_BRANCH}"
    git checkout -B ${RELEASE_BRANCH} ${UPSTREAM_REMOTE}/${MINOR_BRANCH} >/dev/null
else
    echo "New release ${RELEASE_VERSION%.*} detected: creating ${RELEASE_BRANCH} from ${UPSTREAM_REMOTE}/${DEFAULT_BRANCH}"
    git checkout ${DEFAULT_BRANCH}
    git reset --hard ${UPSTREAM_REMOTE}/${DEFAULT_BRANCH}
    git checkout -B ${RELEASE_BRANCH} ${UPSTREAM_REMOTE}/${DEFAULT_BRANCH} >/dev/null
fi

# HACK:! this is temporary to disable the upload to homebrew when we do our testing
# need to find a way how to do this automatically (push to upstream/revert and
# then being able to cherry-pick if $PUSH_REMOTE != 'upstream/origin'?)
[[ ${PUSH_REMOTE} != origin* && ${PUSH_REMOTE} != upstream* ]] && {
    echo "Testing mode detected Cherry-picking commit: 052b0b4ce989fe9aee01027e67e61538b48e1179"
    git cherry-pick 052b0b4ce989fe9aee01027e67e61538b48e1179 >/dev/null
}

COMMITS=$(git log --reverse --no-merges --pretty=format:'%H' ${prev_tag}..HEAD)

echo "Creating changelog for ${RELEASE_VERSION} from ${prev_tag}..HEAD"

changelog=""
for c in ${COMMITS};do
    pr=$(curl -s "https://api.github.com/search/issues?q=${c}" |jq -r '.items[0].html_url')
    pr=$(echo ${pr}|sed 's,https://github.com/tektoncd/cli/pull/,tektoncd/cli#,')
    changelog+="${pr} | $(git log -1 --date=format:'%Y/%m/%d-%H:%M' --pretty=format:'[%an] %s | %cd' ${c})
"
done

echo "${changelog}"

# Add our VERSION so Makefile will pick it up when compiling
echo ${RELEASE_VERSION#v} > VERSION
git add VERSION
git commit -sm "New version ${RELEASE_VERSION}" -m "${changelog}" VERSION
git tag --sign -m \
    "New version ${RELEASE_VERSION}" \
    -m "${changelog}" --force ${RELEASE_VERSION}

git push --force ${PUSH_REMOTE} ${RELEASE_VERSION}
git push --force ${PUSH_REMOTE} ${RELEASE_BRANCH}

echo "Checkout to ${DEFAULT_BRANCH} to use the release pipeline"
git reset --hard ${UPSTREAM_REMOTE}/${DEFAULT_BRANCH}

kubectl create namespace ${TARGET_NAMESPACE} 2>/dev/null || true

for task in ${CATALOG_TASKS};do
  if [ ${task} == "lint" ]; then
    tkn -n ${TARGET_NAMESPACE} hub install task golangci-lint --type artifact --from tekton-catalog-tasks
  else
    tkn -n ${TARGET_NAMESPACE} hub install task golang-${task} --type artifact --from tekton-catalog-tasks
  fi
done

tkn -n ${TARGET_NAMESPACE} hub install task git-clone --type artifact --from tekton-catalog-tasks
tkn -n ${TARGET_NAMESPACE} hub install task goreleaser --type artifact --from tekton-catalog-tasks

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
