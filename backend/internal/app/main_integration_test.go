//go:build integration

package app

import (
	"os"
	"testing"

	"github.com/apotourlyan/ludustudii/internal/testkit"
)

func TestMain(m *testing.M) { os.Exit(testkit.Run(m)) }
