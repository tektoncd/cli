# TektonCD CLI Release Process

- You need to be a member of tektoncd-cli [OWNERS](OWNERS) and be a member of the [CLI maintainers team](https://github.com/orgs/tektoncd/teams/cli-maintainers) to be able to push a new branch for the release and update the CLI Homebrew formula. 

- You will need to have a [GPG key set up](https://help.github.com/en/github/authenticating-to-github/managing-commit-signature-verification) with your git account to push the release branch.

- You need to have your own Kubernetes cluster with Tekton installed under `minikube`, `gcp`, or other means.

- Start reading this [README](tekton/README.md), and you can start a new release
  with the [release.sh](tekton/release.sh) script.

- Update the version numbers in the main [README.md](README.md) to the version you are releasing by opening a pull request to the master branch of this repository. Do not worry about updating the README for the release branch.

- When the release is done in https://github.com/tektoncd/cli/releases, edit the
  release and change it from pre-release as released.

- When it's released, you can build the rpm according to this
  [README](tekton/rpmbuild/README.md). Make sure you have done the PREREQ to be
  able to upload it to the copr repository. This could take some time if copr
  builder is busy.

- When it's released, you can build the rpm according to this
  [README](tekton/debbuild/README.md). Make sure you have done the prerequist to
  be able to upload to the [launchpad team](https://launchpad.net/~tektoncd),
  and make sure as well that you have setup your GPG key and add it to your
  profile on launchpad.

- You need to edit the Changelog according to the other templates in there.
  Usually you may want to do this in hackmd so you can have other cli
  maintainers do this with you. You can see an example
  [here](https://gist.github.com/chmouel/8a837af3a592df47db9e81da8846c673).

- Get someone with OSX to test the `brew install tektoncd/cli/tektoncd-cli`

- Get someone with Fedora to test the RPM:

```shell
dnf makecache
dnf upgrade tektoncd-cli
```

- Make an update for [homebrew-core](https://github.com/Homebrew/homebrew-core/blob/master/Formula/tektoncd-cli.rb) for tektoncd-cli formula, and make sure you get the proper `sha256sum` and update it like this:

  https://github.com/Homebrew/homebrew-core/pull/46492

- Make an update to the `test-runner` and `tkn` image in the [plumbing](https://github.com/tektoncd/plumbing/) repo. The test-runner image is where we run the CI on pipeline which uses `tkn`:

  * [test-runner image](https://github.com/tektoncd/plumbing/blob/master/tekton/images/test-runner/Dockerfile#L51)

  * [tkn image](https://github.com/tektoncd/plumbing/blob/master/tekton/images/tkn/Dockerfile#L17)

- Announce and spread the love on twitter. Make sure you tag
  [@tektoncd](https://twitter.com/tektoncd) account so you get retweeted and
  perhaps add the major new features in the tweet. See [here](https://twitter.com/chmouel/status/1177172542144036869) for an example.
  Do not fear to add a bunch of  emojis 🎉🥳. Ask @vdemeester for tips 🤣.

- Notify the cli channel on slack.
