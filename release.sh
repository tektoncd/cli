RELEASE_VERSION="v0.41.0"
PUSH_REMOTE=upstream
# changelog=""
# for c in ${COMMITS}; do
#     pr=$(curl -s "https://api.github.com/search/issues?q=${c}" | jq -r '.items[0].html_url')
#     pr=$(echo ${pr} | sed 's,https://github.com/tektoncd/cli/pull/,tektoncd/cli#,')
#     changelog+="${pr} | $(git log -1 --date=format:'%Y/%m/%d-%H:%M' --pretty=format:'[%an] %s | %cd' ${c})
# "
# done

# Add our VERSION so Makefile will pick it up when compiling
# echo ${RELEASE_VERSION#v} >VERSION
# git add VERSION
# git commit -sm "New version ${RELEASE_VERSION}" -m "${changelog}" VERSION
# git tag --sign -m \
#     "New version ${RELEASE_VERSION}" \
#     -m "${changelog}" --force ${RELEASE_VERSION}

# git push --force ${PUSH_REMOTE} ${RELEASE_VERSION}
# git push --force ${PUSH_REMOTE} release-${RELEASE_VERSION}

# kubectl create namespace ${TARGET_NAMESPACE} 2>/dev/null || true

# for task in ${CATALOG_TASKS}; do
#     if [ ${task} == "lint" ]; then
#         tkn -n ${TARGET_NAMESPACE} hub install task golangci-lint
#     else
#         tkn -n ${TARGET_NAMESPACE} hub install task golang-${task}
#     fi
# done

# tkn -n ${TARGET_NAMESPACE} hub install task git-clone
# tkn -n ${TARGET_NAMESPACE} hub install task goreleaser

# kubectl -n ${TARGET_NAMESPACE} apply -f ./tekton/get-version.yaml
# kubectl -n ${TARGET_NAMESPACE} apply -f ./tekton/publish.yaml

# if ! kubectl -n ${TARGET_NAMESPACE} get secret ${SECRET_NAME} -o name >/dev/null 2>/dev/null; then
#     github_token=$(git config --get github.oauth-token || true)

#     [[ -z ${github_token} ]] && {
#         read -e -p "Enter your Github Token: " github_token
#         [[ -z ${github_token} ]] && {
#             echo "no token provided"
#             exit 1
#         }
#     }

#     kubectl -n ${TARGET_NAMESPACE} create secret generic ${SECRET_NAME} --from-literal=bot-token=${github_token}
# fi

# kubectl -n ${TARGET_NAMESPACE} apply -f ./tekton/release-pipeline.yml

# sleep 2

URL=$(git remote get-url "${PUSH_REMOTE}" | sed 's,git@github.com:,https://github.com/,')
# # Start the pipeline
tkn -n release pipeline start cli-release-pipeline \
    -p revision="${RELEASE_VERSION}" \
    -p url="${URL}" \
    -w name=shared-workspace,volumeClaimTemplateFile=./tekton/volume-claim-template.yml

# tkn -n ${TARGET_NAMESPACE} pipeline logs cli-release-pipeline -f --last
