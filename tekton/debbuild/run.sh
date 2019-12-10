#!/bin/bash
set -eux

# Uploading to tektoncd/cli ppa :
# https://launchpad.net/~tektoncd/+archive/ubuntu/cli
# You need to join this team to be able to upload there
# and have your gpg setup in your profile
PPATARGET=tektoncd/cli
GPG_KEY=${GPG_KEY}

[[ -n ${GPG_KEY} ]] || {
	echo "You need to setup your GPG_KEY"
	exit 1
}


gpg --list-secret-keys >/dev/null || { echo "You need to have a secret GPG key"; exit 1 ;}

cd container
docker build -t ubuntu-build-deb .

cd ..
fpath=$(readlink -f control)
docker run --rm \
	 -v ~/.gnupg:/root/.gnupg \
	 -v ${fpath}:/debian --name ubuntu-build-deb \
	 --env PPATARGET=${PPATARGET} \
	 --env GPG_KEY=${GPG_KEY} \
	 -it ubuntu-build-deb \
	/run.sh
