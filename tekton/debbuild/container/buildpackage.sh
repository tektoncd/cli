#!/bin/bash
set -eux
# Increase this on minor *package* releases (ie fix on the packaging),
# decrease it to 0 on major *package* release
# (need ot find a better way to do that)
RELEASE=0

[[ -z ${GPG_KEY} ]] && {
    echo "You need to setup your GPG_KEY."
    exit 1
}

[[ -z ${PPATARGET} ]] && {
    echo "You need to setup a PPA Target."
    exit 1
}


TMPD=$(mktemp -d)
mkdir -p ${TMPD}
clean() { rm -rf ${TMPD} ;}
trap clean EXIT
# TMPD=/tmp/debug;mkdir -p ${TMPD}

curl -o ${TMPD}/output.json -s https://api.github.com/repos/tektoncd/cli/releases/latest
version=$(python3 -c "import sys, json;x=json.load(sys.stdin);print(x['tag_name'])" < ${TMPD}/output.json)
version=${version/v}

cd ${TMPD}

curl -C- -L -# -o tektoncd-cli_${version}.orig.tar.gz -L https://github.com/tektoncd/cli/archive/v${version}.tar.gz
tar xzf ${TMPD}/tektoncd-cli_${version}.orig.tar.gz

mv cli-${version} tektoncd-cli-${version}
cd tektoncd-cli-${version}

# Make it easy for devs
[[ -d /debian ]] && { rm -rf debian && cp -a /debian . ; }

# debian/rules will use it automatically to set the binary version
echo ${version} > debian/VERSION

dch -M -v ${version}-${RELEASE} -D $(sed -n '/DISTRIB_CODENAME/ { s/.*=//;p;;}' /etc/lsb-release) "new update"

pcscd
gpgconf --kill gpg-agent && gpg-agent --pinentry-program /usr/bin/pinentry-curses --verbose --daemon
gpg --card-status || true
debuild -S --force-sign -k${GPG_KEY}

cd ..

dput ppa:${PPATARGET}  *source.changes
