#!/bin/bash
set -e

echo "Installing Go tools..."
go install github.com/air-verse/air@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install golang.org/x/tools/cmd/stringer@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
# Pin the standalone binary; keep lint tooling out of the backend module.
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/v2.15.0/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v2.15.0
golangci-lint version
echo "Go tools installed successfully!"

echo "Installing frontend dependencies..."
(cd frontend && npm install)
echo "Frontend dependencies installed successfully!"

echo "Installing PostgreSQL client..."
apt-get update && apt-get install -y postgresql-client
echo "PostgreSQL client installed successfully!"
