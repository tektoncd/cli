# using rolling to get latest golang
FROM ubuntu:rolling

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
			   bash-completion \
			   golang-any \
               devscripts \
			   debhelper \
			   dh-golang \
               fakeroot \
    && apt-get clean \
    && rm -rf /tmp/* /var/tmp/*

ADD buildpackage.sh /run.sh
