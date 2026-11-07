#!/bin/bash
set -e

echo "Installing Go tools..."
# Install the versions tracked by backend/tools/go.mod for editor and PATH access.
go -C backend/tools install \
  github.com/air-verse/air \
  github.com/go-delve/delve/cmd/dlv \
  golang.org/x/tools/cmd/stringer \
  golang.org/x/vuln/cmd/govulncheck \
  github.com/pressly/goose/v3/cmd/goose
# The Makefile holds the golangci-lint pin; CI installs it the same way.
make install-golangci-lint
echo "Go tools installed successfully!"

echo "Installing frontend dependencies..."
(cd frontend && npm install)
echo "Frontend dependencies installed successfully!"

echo "Installing PostgreSQL client..."
apt-get update && apt-get install -y postgresql-client
echo "PostgreSQL client installed successfully!"
