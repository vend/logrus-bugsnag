#! /bin/bash

# Install and run golangci-lint v1.17.1.
# Last golangci-lint version is available here: https://github.com/golangci/golangci-lint/releases
# Linter will display error for information, but will not make the build fail.

curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(go env GOPATH)/bin v1.17.1
golangci-lint run
