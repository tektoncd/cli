# using rolling to get latest golang
FROM ubuntu:rolling

ENV DEBIAN_FRONTEND=noninteractive

RUN set -ex \
    && sed -i -- 's/# deb-src/deb-src/g' /etc/apt/sources.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
    build-essential \
    dput \
    cdbs \
    git \
    curl \
    equivs \
    vim \
    libdistro-info-perl \
    golang-any \
    devscripts \
    debhelper \
    dh-golang \
    fakeroot \
    pcscd \
    scdaemon \
    && apt-get clean \
    && rm -rf /tmp/* /var/tmp/*

ADD buildpackage.sh /run.sh
