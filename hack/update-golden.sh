#!/usr/bin/env bash
# This will run `go test a/package -test.update-golden=true` on all packages that are importing `gotest.tools/golden`
# This will update the golden files with the current output.
# Run this only when you are sure the output is meant to change.
go test $(go list -f '{{ .ImportPath }} {{ .TestImports }}' ./... | grep gotest.tools/golden | awk '{print $1}' | tr '\n' ' ') -test.update-golden=true
