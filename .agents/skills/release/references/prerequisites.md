# Release Prerequisites Reference

## Required Tools

| Tool | Purpose | Install |
| --- | --- | --- |
| `kubectl` | Kubernetes cluster access | https://kubernetes.io/docs/tasks/tools/ |
| `jq` | JSON processing in release script | `brew install jq` / `dnf install jq` |
| `tkn` | Tekton CLI (for pipeline management) | https://tekton.dev/docs/cli/ |
| `git` | Version control, tagging, branching | System package manager |
| `gh` | GitHub CLI (for release notes, auth) | https://cli.github.com/ |

## Access Requirements

| Requirement | How to get it |
| --- | --- |
| CLI maintainers team | Request access via [tektoncd/cli OWNERS](https://github.com/tektoncd/cli/blob/main/OWNERS) |
| GitHub PAT (`admin:org, read:packages, repo, write:packages`) | https://github.com/settings/tokens |
| GPG signing key in git | https://help.github.com/en/github/authenticating-to-github/managing-commit-signature-verification |
| Kubernetes cluster with Tekton | minikube, GKE, or any cluster with `kubectl get pipeline` working |
| Copr admin access (RPM) | Request at https://copr.fedorainfracloud.org/coprs/chmouel/tektoncd-cli/permissions/ |
| Launchpad team (DEB) | Join at https://launchpad.net/~tektoncd/+join and upload GPG key to profile |

## Environment

| Variable | Required | Purpose |
| --- | --- | --- |
| `GOPATH` | Yes | release.sh uses `${GOPATH}/src/github.com/tektoncd/cli` |
| `PUSH_REMOTE` | No | Override push remote for testing (default: `upstream`) |
| `GPG_KEY` | For DEB builds | GPG key user ID for signing Debian packages |

## Release Branch Convention

Release branches follow `release-vX.Y.x`:

- `release-v0.46.x` for the v0.46 release line
- `release-v0.43.x` for the v0.43 release line

### Minor release (e.g., v0.46.0)

- Script creates `release-vX.Y.x` from `main`
- No prior branch expected

### Patch release (e.g., v0.43.3)

- Branch `release-vX.Y.x` must already exist
- All fixes must be merged into the `.x` branch before running the release
- Script detects the branch exists and checks it out

## Previous Tag Detection

The release script auto-detects the previous tag for changelog generation:

- **Patch release**: Finds the latest `vX.Y.*` tag within the same minor version (e.g., for `v0.43.3`, finds `v0.43.2`)
- **Minor release**: Finds the latest `vX.Y.Z` tag across all versions (e.g., for `v0.46.0`, finds `v0.45.0`)
