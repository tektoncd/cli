ARG GOLANG_VERSION=1.25.6
ARG DEBIAN_VERSION=12

FROM golang:${GOLANG_VERSION} AS builder
ARG RELEASE_VERSION=
COPY . /go/src/github.com/tektoncd/cli
WORKDIR /go/src/github.com/tektoncd/cli
RUN make RELEASE_VERSION=${RELEASE_VERSION} bin/tkn

FROM debian:${DEBIAN_VERSION} AS tkn
COPY --from=builder /go/src/github.com/tektoncd/cli/bin/tkn /usr/bin
