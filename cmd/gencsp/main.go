// Command gencsp is the codegen stage of the CSP pipeline: it reads the
// committed DDF snapshots (metadata/csp) and emits the generated SDK surface
// (windowscsp/csp, windowscsp/policy, windowscsp/registry.go). It is offline
// and deterministic; CI regenerates and diffs against the committed tree.
//
//	go run ./cmd/gencsp
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/deploymenttheory/go-sdk-windowscsp/internal/codegen"
)

const modulePath = "github.com/deploymenttheory/go-sdk-windowscsp/windowscsp"

func main() {
	metadataDir := flag.String("metadata", filepath.Join("metadata", "csp"), "snapshot input directory")
	outDir := flag.String("out", "windowscsp", "output directory (the windowscsp package root)")
	flag.Parse()

	if err := codegen.Run(*metadataDir, *outDir, modulePath); err != nil {
		fmt.Fprintln(os.Stderr, "gencsp:", err)
		os.Exit(1)
	}
}
