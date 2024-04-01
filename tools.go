//go:build tools

package main

// This import is required to make sure go mod tidy doesn't remove packages that make pact-install depends on
import (
	_ "github.com/pact-foundation/pact-go/v2"
)
