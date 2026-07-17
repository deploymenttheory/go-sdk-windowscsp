// AUTHOR a SyncML batch document that starts a Microsoft Defender scan —
// this example's product is the document, not an executed change. It also
// shows that Exec nodes carrying a payload take it as a typed argument.
//
// syncml.Recorder is the OFFLINE authoring transport: calls are queued,
// never executed, and rec.Document() emits the <SyncBody> batch for an MDM
// to deliver. To EXECUTE against a live SyncML endpoint with SyncML handled
// invisibly, use syncml.Executor instead (see
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

	// Scan payload per the Defender CSP docs: "1" = quick scan, "2" = full scan.
	if err := c.CSP.Defender.ExecScan(context.Background(), "1"); err != nil {
		log.Fatalf("queue scan: %v", err)
	}

	fmt.Println("SyncML batch to deliver to the device:")
	fmt.Println(rec.Document())
}
