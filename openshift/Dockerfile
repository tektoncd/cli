FROM openshift/origin-release:golang-1.12

# download, verify and install openshift client tools (oc and kubectl)
WORKDIR /tmp
RUN OPENSHIFT_CLIENT_VERSION=$(curl -s https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/release.txt|sed -n '/Version:/ { s/[ ]*Version:[ ]*// ;p}') \
    && curl -L -O -s https://mirror.openshift.com/pub/openshift-v4/clients/ocp/$OPENSHIFT_CLIENT_VERSION/openshift-client-linux-$OPENSHIFT_CLIENT_VERSION.tar.gz \
    && curl -L -s https://mirror.openshift.com/pub/openshift-v4/clients/ocp/$OPENSHIFT_CLIENT_VERSION/sha256sum.txt | \
       grep openshift-client-linux-$OPENSHIFT_CLIENT_VERSION.tar.gz > openshift-origin-client-tools.sha256 \
    && sha256sum -c openshift-origin-client-tools.sha256 \
    && mkdir openshift-origin-client-tools \
    && tar xzf openshift-client-linux-$OPENSHIFT_CLIENT_VERSION.tar.gz --directory openshift-origin-client-tools \
    && mv /tmp/openshift-origin-client-tools/oc /usr/bin/oc \
    && mv /tmp/openshift-origin-client-tools/kubectl /usr/bin/kubectl \
    && rm -rf ./openshift* \
    && oc version

# Allow runtime users to add entries to /etc/passwd
RUN chmod g+rw /etc/passwd
