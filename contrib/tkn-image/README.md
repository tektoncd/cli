# tkn oci image

This directory contains the `Dockerfile` used to build the official
`tkn` image. It is a multi-stage `Dockerfile` that defines 1 targets:

```
docker build --file=contrib/tkn-image/Dockerfile \
    --target=tkn --tag tkn:latest .
```
