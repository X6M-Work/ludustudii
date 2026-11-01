#!/bin/bash
set -e

echo "Installing Go tools..."
# Install the versions tracked by backend/go.mod for editor and PATH access.
go -C backend install \
  github.com/air-verse/air \
  github.com/go-delve/delve/cmd/dlv \
  golang.org/x/tools/cmd/stringer \
  github.com/pressly/goose/v3/cmd/goose
# Pin the standalone binary; keep lint tooling out of the backend module.
GOLANGCI_VERSION=v2.15.0
curl -sSfL "https://raw.githubusercontent.com/golangci/golangci-lint/${GOLANGCI_VERSION}/install.sh" | sh -s -- -b "$(go env GOPATH)/bin" "$GOLANGCI_VERSION"
golangci-lint version
echo "Go tools installed successfully!"

echo "Installing frontend dependencies..."
(cd frontend && npm install)
echo "Frontend dependencies installed successfully!"

echo "Installing PostgreSQL client..."
apt-get update && apt-get install -y postgresql-client
echo "PostgreSQL client installed successfully!"
