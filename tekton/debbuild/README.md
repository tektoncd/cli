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
