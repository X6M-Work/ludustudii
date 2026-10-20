#!/bin/bash
set -e

echo "Installing Go tools..."

go install github.com/air-verse/air@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install golang.org/x/tools/cmd/stringer@latest

echo "Go tools installed successfully!"

cd frontend && npm install