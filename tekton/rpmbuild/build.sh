#!/usr/bin/env bash

currentdir=$(dirname $(readlink -f $0))
repospecfile=${currentdir}/tekton.spec
TMPD="/tmp/cli"
mkdir -p ${TMPD}

set -ex

[[ -z ${GITHUB_TOKEN} ]] && { echo "We need a GITHUB_TOKEN".; exit 1;}

curl -H "Authorization: token ${GITHUB_TOKEN}" -o ${TMPD}/output.json -s https://api.github.com/repos/tektoncd/cli/releases/latest
version=$(python3 -c "import sys, json;x=json.load(sys.stdin);print(x['tag_name'])" < ${TMPD}/output.json)
version=${version/v}

sed "s/_VERSION_/${version}/" ${repospecfile} > ${TMPD}/tekton.spec

sed -i '/bundled(golang/d' ${TMPD}/tekton.spec

mapfile -t bundle < <(python3 -c "ver = None;
def version(line): global ver; ver = line.split()[2]; return '';

print '\n'.join(filter(None,[line for line in ['Provides: bundled(golang({})) = {}'.format(line.rstrip('\n'), ver) if line[0] != '#' else version(line.rstrip('\n').replace('+incompatible','')) for line in open('../../vendor/modules.txt')]][::-1]))")

for i in "${bundle[@]}"
do
   sed -i "/vendored\slibraries/a $i" ${TMPD}/tekton.spec
done

( cd ${currentdir}/../../docs/man && tar czf ${TMPD}/manpages.tar.gz  man1)

cd ${TMPD}

# TODO: multiarch
tarball=tkn_${version}_Linux_x86_64.tar.gz
[[ -e ${tarball} ]] || \
    curl -f -H "Authorization: token ${GITHUB_TOKEN}" \
        -O -L https://github.com/tektoncd/cli/releases/download/v${version}/${tarball}

rpmbuild -bs tekton.spec --define "_sourcedir $PWD" --define "_srcrpmdir $PWD"

copr-cli --config=<( echo "${COPR_TOKEN}" ) build tektoncd-cli tektoncd-cli-${version}-1.src.rpm
