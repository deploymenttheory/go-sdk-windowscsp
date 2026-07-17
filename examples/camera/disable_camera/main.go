// AUTHOR a SyncML batch document that disables the device camera — this
// example's product is the document, not an executed change.
//
// syncml.Recorder is the OFFLINE authoring transport: the typed call below
// is queued, never executed, and rec.Document() emits the <SyncBody> batch
// to paste into an Intune custom OMA-URI profile or hand to an MDM delivery
// pipeline. Nothing answers here, so a Recorder is write-only.
//
// To EXECUTE calls against a live SyncML endpoint instead — with reads
// returning typed values and SyncML handled invisibly — use syncml.Executor
// (see examples/transport/syncml_executor).
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/policy/camera"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/syncml"
)

func main() {
	rec := syncml.NewRecorder()
	c := windowscsp.NewClient(rec)

	// Policy areas live under ./Device/Vendor/MSFT/Policy/Config; allowed
	// values come from the DDF as typed constants.
	err := c.Policy.Camera.UpdateAllowCamera(context.Background(), camera.AllowCameraNotAllowed)
	if err != nil {
		log.Fatalf("queue AllowCamera: %v", err)
	}

	fmt.Println("OMA-URI:", camera.URIAllowCamera)
	fmt.Println(rec.Document())
}
