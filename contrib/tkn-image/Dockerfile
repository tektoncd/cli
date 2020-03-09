ARG GOLANG_VERSION=1.13.8
ARG DEBIAN_VERSION=10

FROM golang:${GOLANG_VERSION} as builder
COPY . /go/src/github.com/tektoncd/cli
WORKDIR /go/src/github.com/tektoncd/cli
RUN make bin/tkn

FROM debian:${DEBIAN_VERSION} as tkn
COPY --from=builder /go/src/github.com/tektoncd/cli/bin/tkn /usr/bin
