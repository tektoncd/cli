---
name: release
description: This skill should be used when the user asks to "do a release", "cut a release", "create a patch release", "create a minor release", "release v0.X.Y", or wants to perform the Tekton CLI release process. Guides the user through the full release workflow including prerequisites, running release.sh, and post-release steps (rpm, deb, homebrew, plumbing).
version: 0.1.0
---

# Tekton CLI Release

Guide the user through the complete Tekton CLI release process — from prerequisites through post-release packaging and distribution.

## Purpose

Skill that:

- Validates all prerequisites before starting
- Determines whether this is a minor or patch release
- Guides running `tekton/release.sh`
- Walks through all post-release steps in order
- Tracks progress through the checklist

## Workflow

### Step 1: Determine release type and version

Ask the user for the target release version if not already provided. The version must match `vX.Y.Z` (e.g., `v0.46.0`, `v0.43.3`).

Determine the release type:

- **Minor release** (`Z == 0`, e.g., `v0.46.0`): Creates a new `release-vX.Y.x` branch from `main`.
- **Patch release** (`Z > 0`, e.g., `v0.43.3`): Uses the existing `release-vX.Y.x` branch. All fixes must already be merged into the `.x` branch.

Tell the user which type was detected.

### Step 2: Validate prerequisites

Check each prerequisite and report status. Refer to `references/prerequisites.md` for the full list.

**Automated checks** (run these):

```bash
# Required binaries
for bin in kubectl jq tkn git gh; do
  command -v "$bin" >/dev/null 2>&1 && echo "✓ $bin" || echo "✗ $bin missing"
done

# Kubernetes cluster access
kubectl version --short 2>/dev/null && echo "✓ cluster accessible" || echo "✗ no cluster access"

# Tekton installed on cluster
kubectl get pipeline 2>/dev/null && echo "✓ Tekton installed" || echo "✗ Tekton not installed"

# GPG key configured
git config user.signingkey && echo "✓ GPG signing key set" || echo "✗ no GPG signing key"

# GOPATH set and project path correct
echo "GOPATH=$GOPATH"
[[ -d "${GOPATH}/src/github.com/tektoncd/cli" ]] && echo "✓ project at GOPATH path" || echo "✗ project not at \${GOPATH}/src/github.com/tektoncd/cli"

# gh authenticated
gh auth status 2>&1

# Clean working directory
git status --porcelain
```

**Manual checks** (remind the user to verify):

- Member of [CLI maintainers team](https://github.com/orgs/tektoncd/teams/cli-maintainers)
- GitHub personal access token with `admin:org, read:packages, repo, write:packages` scopes
- Access to [copr repository](https://copr.fedorainfracloud.org/coprs/chmouel/tektoncd-cli/) (for RPM)
- Member of [launchpad team](https://launchpad.net/~tektoncd) with GPG key uploaded (for DEB)

If any automated check fails, stop and help resolve it before proceeding.

**CRITICAL**: The working directory must be clean (no uncommitted changes). If `git status --porcelain` returns output, tell the user to stash or commit changes first.

### Step 3: Patch release — verify fixes are merged

**Skip this step for minor releases.**

For patch releases, verify the release branch exists and contains the expected fixes:

```bash
RELEASE_BRANCH="release-v${VERSION%.*}.x"
git fetch -a --tags upstream
git ls-remote --exit-code upstream "refs/heads/${RELEASE_BRANCH}"
```

Show recent commits on the release branch so the user can confirm the right fixes are included:

```bash
git log --oneline upstream/${RELEASE_BRANCH} -20
```

Ask: **"Are all the fixes for this patch release merged into `${RELEASE_BRANCH}`? (y/n)"**

Do not proceed until confirmed.

### Step 4: Run the release script

**CRITICAL**: This step requires the user to run the command themselves since `release.sh` is interactive (it may prompt for a GitHub token).

Tell the user to run:

```bash
cd ${GOPATH}/src/github.com/tektoncd/cli
./tekton/release.sh vX.Y.Z
```

Explain what the script does:

1. Fetches tags and determines previous release tag
2. For minor releases: creates `release-vX.Y.x` branch from `main`
3. For patch releases: checks out existing `release-vX.Y.x` branch
4. Generates a changelog from commits between previous tag and HEAD
5. Updates the `VERSION` file, commits, and creates a signed tag
6. Pushes the tag and release branch to `upstream`
7. Installs Tekton catalog tasks on the cluster
8. Applies the release pipeline and triggers it
9. Streams pipeline logs via `tkn`

Tell the user: **"Run the release script and let me know when it completes successfully, or if you hit any errors."**

Wait for the user to report back before continuing.

### Step 5: Mark release as published on GitHub

After the release pipeline completes, the GitHub release will be in pre-release state.

```bash
gh release view vX.Y.Z --json isDraft,isPrerelease,tagName
```

Tell the user to:

1. Go to https://github.com/tektoncd/cli/releases/tag/vX.Y.Z
2. Edit the release
3. Change it from pre-release to released (uncheck "Set as a pre-release")
4. Publish the release

**IMPORTANT**: This must be done before building the RPM package.

Ask the user to confirm the release is published before proceeding.

### Step 6: Generate release notes

Invoke the `release-notes` skill to generate and publish release notes for the tag. Tell the user:

**"Let's generate the release notes. You can run `/release-notes vX.Y.Z` or I can generate them now."**

### Step 7: Build RPM package

Guide the user through building the RPM package per `tekton/rpmbuild/README.md`.

**Prerequisites** (remind user):

- Access to [copr repository](https://copr.fedorainfracloud.org/coprs/chmouel/tektoncd-cli/)
- Copr API config at `~/.config/copr` (get from https://copr.fedorainfracloud.org/api/)
- Change `username` field in copr config to `chmouel`

**Steps**:

```bash
# Create copr secret if not exists
kubectl -n release get secret copr-cli-config || \
  kubectl -n release create secret generic copr-cli-config --from-file=copr=${HOME}/.config/copr

# Ensure git-clone task is installed
tkn -n release task list | grep git-clone || tkn -n release hub install task git-clone

# Create and run the RPM build
kubectl -n release apply -f tekton/rpmbuild/rpmbuild.yml
kubectl -n release create -f tekton/rpmbuild/rpmbuild-run.yml

# Watch logs
tkn -n release pipelinerun logs rpmbuild-pipelinerun -f
```

Tell the user this may take time if the copr builder is busy.

### Step 8: Build Debian package

Guide the user through building the DEB package per `tekton/debbuild/README.md`. This can start while the RPM is building.

**Prerequisites** (remind user):

- Member of [launchpad team](https://launchpad.net/~tektoncd)
- GPG key uploaded to launchpad profile
- Set `GPG_KEY` environment variable to GPG key user ID

**Steps**:

```bash
export GPG_KEY=<user-gpg-key-id>
cd tekton/debbuild
./run.sh
```

The build uploads to https://launchpad.net/~tektoncd/+archive/ubuntu/cli/+packages — user can check build logs there.

**Known issue**: If the build fails with `build flag -mod=vendor only valid when using modules`, edit `tekton/debbuild/control/rules` and append `GO111MODULE=on` at the build line. If re-pushing after a failed build, increment the release version in `tekton/debbuild/container/buildpackage.sh`.

### Step 9: Homebrew update

Tell the user:

- Homebrew Core has a GitHub Action that automatically bumps the formula every ~3 hours
- Check that a PR like [this example](https://github.com/Homebrew/homebrew-core/pull/171551) is created and merged
- Alternatively, manually update [homebrew-core](https://github.com/Homebrew/homebrew-core) for `tektoncd-cli` formula

### Step 10: Update plumbing repo (optional)

**This step is optional.** Mention it to the user but do not block on it.

Tell the user they can optionally update the `tkn` version in the [tektoncd/plumbing](https://github.com/tektoncd/plumbing/) repo:

1. **test-runner image**: Update `ARG TKN_VERSION=<NEW_VERSION>` in [test-runner Dockerfile](https://github.com/tektoncd/plumbing/blob/main/tekton/images/test-runner/Dockerfile)
2. **tkn image**: Update `ARG TKN_VERSION=<NEW_VERSION>` in [tkn Dockerfile](https://github.com/tektoncd/plumbing/blob/main/tekton/images/tkn/Dockerfile)

### Step 11: Update Arch Linux

Tell the user to go to https://archlinux.org/packages/extra/x86_64/tekton-cli/flag/ and notify packagers that a new version is available, leaving the release URL and their email address.

### Step 12: Update README version

Remind the user to update version numbers in the main `README.md` via a PR to `main`.

## Progress Tracking

After each step, confirm with the user before moving on. Use a checklist format:

```text
Release vX.Y.Z Progress:
  [✓] 1. Release type determined (minor/patch)
  [✓] 2. Prerequisites validated
  [✓] 3. Fixes verified (patch only)
  [✓] 4. release.sh completed
  [ ] 5. GitHub release published
  [ ] 6. Release notes generated
  [ ] 7. RPM package built
  [ ] 8. DEB package built
  [ ] 9. Homebrew updated
  [ ] 10. Plumbing repo updated (optional)
  [ ] 11. Arch Linux notified
  [ ] 12. README version updated
```

## Error Handling

| Scenario | Action |
| --- | --- |
| Binary missing | Tell user how to install it |
| No cluster access | Tell user to configure kubeconfig |
| Tekton not installed | Link to Tekton Pipelines install docs |
| No GPG key | Link to GitHub GPG setup guide |
| Dirty working directory | Tell user to stash or commit |
| Release branch missing (patch) | Error — fixes need to be merged first |
| release.sh fails | Help debug based on error output |
| RPM build fails | Check copr builder status and logs |
| DEB build fails | Check launchpad build logs, suggest GO111MODULE fix |
| GitHub release not found | Script may not have pushed; check tags |

## User Confirmation Requirements

**CRITICAL**: Never run `release.sh` directly — it is interactive and must be run by the user. Always confirm after each major step before proceeding. Never push tags or branches without the user having explicitly run the release script.
