// AUTHOR a SyncML batch document that triggers an immediate reboot via the
// Reboot CSP — this example's product is the document, not an executed
// change.
//
// syncml.Recorder is the OFFLINE authoring transport: the Exec call below
// is queued, never executed, and rec.Document() emits the <SyncBody> batch
// for an MDM to deliver. To EXECUTE against a live SyncML endpoint with
// SyncML handled invisibly, use syncml.Executor instead (see
// examples/transport/syncml_executor).
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/syncml"
)

func main() {
	rec := syncml.NewRecorder()
	c := windowscsp.NewClient(rec)

	// RebootNow triggers a reboot within 5 minutes of delivery.
	if err := c.CSP.Reboot.ExecRebootNow(context.Background()); err != nil {
		log.Fatalf("queue RebootNow: %v", err)
	}

	fmt.Println("SyncML batch to deliver to the device:")
	fmt.Println(rec.Document())
}
