// Command bundlespec writes the OpenAPI specification bundled into a single
// JSON document to the file named by its argument.
//
// Usage: go run ./cmd/tools/bundlespec <output file>
package main

import (
	"fmt"
	"os"

	"github.com/apotourlyan/ludustudii/apispec"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: bundlespec <output file>")
		os.Exit(2)
	}
	b, err := apispec.Bundle()
	if err != nil {
		fmt.Fprintln(os.Stderr, "bundlespec:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(os.Args[1], b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "bundlespec:", err)
		os.Exit(1)
	}
}
