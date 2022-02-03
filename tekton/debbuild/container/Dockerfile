# using rolling to get latest golang
FROM ubuntu:latest

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
    golang-1.14 \
    devscripts \
    debhelper \
    dh-golang \
    fakeroot \
    pcscd \
    scdaemon \
    && apt-get clean \
    && rm -rf /tmp/* /var/tmp/*

ADD buildpackage.sh /run.sh
