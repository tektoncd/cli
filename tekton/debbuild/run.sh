#!/usr/bin/env bash
set -eux

# Uploading to tektoncd/cli ppa :
# https://launchpad.net/~tektoncd/+archive/ubuntu/cli
# You need to join this team to be able to upload there
# and have your gpg setup in your profile
PPATARGET=tektoncd/cli
GPG_KEY=${GPG_KEY}
YUBIKEY=${YUBIKEY:-}
ADDITIONAL_ARGS=${ADDITIONAL_ARGS:-}

[[ -n ${GPG_KEY} ]] || {
    echo "You need to setup your GPG_KEY"
    exit 1
}

[[ -n ${YUBIKEY} ]] && {
    ADDITIONAL_ARGS="${ADDITIONAL_ARGS} -v /dev/bus/usb:/dev/bus/usb \
     -v /sys/bus/usb:/sys/bus/usb \
     -v /sys/devices:/sys/devices \
     -v ${YUBIKEY}:${YUBIKEY} \
     --privileged"
}

gpg --list-secret-keys >/dev/null || { echo "You need to have a secret GPG key"; exit 1 ;}

cd container
docker build -t ubuntu-build-deb .

cd ..
fpath=$(readlink -f control)
docker run --rm \
       -v ~/.gnupg:/root/.gnupg \
       $ADDITIONAL_ARGS \
       -v ${fpath}:/debian --name ubuntu-build-deb \
       --env PPATARGET=${PPATARGET} \
       --env GPG_KEY=${GPG_KEY} \
       -it ubuntu-build-deb \
       /run.sh
