* We are proving a PPA for users to install, it should be compatible with most debian versions.

* Since we need a GPG key we build directly the docker image and mount our ~/.gnupg into the container.

* First you need to join the launchpad team here :

  https://launchpad.net/~tektoncd/+join

* You need to make sure you upload your GPG key to your launchpad profile, ie:

  https://launchpad.net/~chmouel/+editpgpkeys

  (replace chmouel by your user)

* add an environment variable called GPG_KEY with your GPG key user ID

* start `./run.sh`

* It should upload to
  https://launchpad.net/~tektoncd/+archive/ubuntu/cli/+packages you can inspect
  the build logs in there in case of failure.

- **Note**: There can be a case when the uploaded build fails on launchpad with the below error:

  ```
  + go build -mod=vendor -v -o obj-x86_64-linux-gnu/bin/tkn -ldflags -X github.com/tektoncd/cli/pkg/cmd/version.clientVersion=0.25.1 ./cmd/tkn
  build flag -mod=vendor only valid when using modules
  make[1]: *** [debian/rules:17: override_dh_auto_build] Error 1
  make[1]: Leaving directory '/<<PKGBUILDDIR>>'
  make: *** [debian/rules:14: build] Error 2
  dpkg-buildpackage: error: debian/rules build subprocess returned exit status 2
  --------------------------------------------------------------------------------
  ```

  In this case, edit the [rules](./control//rules) file. Try appending `GO111MODULE=on` at Line #20.

- **Note**: If the first build fails, then in order to repush the new build, increment the release
  version in [buildpackage.sh](./container/buildpackage.sh) at Line #6
