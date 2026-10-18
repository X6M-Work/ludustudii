# Variables
BINARY_NAME=bin/app
MAIN_PATH=./cmd/api  # Replace with the path to your main package (or .)

.PHONY: run dev build clean

# Standard run (no air, builds and executes)
run:
	go run $(MAIN_PATH)

# Live reload using air
dev:
	air

# Build binary
build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

# Clean compiled artifacts
clean:
	go clean
	rm -rf bin/ tmp/