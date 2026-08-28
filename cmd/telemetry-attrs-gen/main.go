// Command telemetry-attrs-gen regenerates the client-analytics attribute
// allowlist (TFCA-B2).
//
// It constructs the provider, walks the runtime ResourcesMap, and writes each
// managed resource's sorted, names-only attribute list to a JSON file that is
// committed to the repo. Because it walks the constructed map rather than
// scanning source for a naming convention, resources whose constructor doesn't
// follow the xResource() pattern are covered too. CI regenerates and diffs the
// output so the committed file can never drift from the live schema.
//
// Usage:
//
//	go run ./cmd/telemetry-attrs-gen [-out path]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider"
)

// defaultOutPath is relative to the repository root, which is the working
// directory when the tool is invoked as `go run ./cmd/telemetry-attrs-gen`.
const defaultOutPath = "internal/provider/telemetry_attrs_allowlist.json"

func main() {
	out := flag.String("out", defaultOutPath, "path to write the generated allowlist JSON")
	flag.Parse()

	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "telemetry-attrs-gen:", err)
		os.Exit(1)
	}
}

func run(outPath string) error {
	allowlist := provider.ResourceAttributeAllowlist()

	// json.MarshalIndent sorts map keys, and each attribute slice is already
	// sorted, so the output is deterministic across runs and machines.
	data, err := json.MarshalIndent(allowlist, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling allowlist: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", outPath, err)
	}
	return nil
}
