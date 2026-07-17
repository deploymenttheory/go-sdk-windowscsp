// Work with OMA-URIs directly — both directions:
//
//  1. GENERATE a URI from the SDK. Every generated package exports its
//     OMA-URIs as constants (static nodes) and builder functions (dynamic
//     nodes). This is what you paste into an Intune custom OMA-URI profile,
//     where Intune wants the URI + data type + value rather than SyncML.
//
//  2. OPERATE ON a CUSTOM URI the SDK does not cover — an OEM or
//     third-party CSP, or a node newer than the pinned DDF release. Every
//     transport implements client.Client, whose six verbs take plain URI
//     strings, so the generated services are a convenience layer, never a
//     cage: call the transport directly with any URI and a typed
//     client.Value.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/client"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/csp/vpnv2"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/policy/camera"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/syncml"
)

func main() {
	// --- 1. Generate URIs from the SDK -------------------------------------
	// Static nodes are constants; dynamic nodes are builder functions taking
	// the runtime name. Ready for an Intune custom OMA-URI profile:
	fmt.Println("Intune custom OMA-URI rows generated from the SDK:")
	fmt.Printf("  URI: %s\n  Type: Integer, Value: %d  (disable camera)\n\n",
		camera.URIAllowCamera, camera.AllowCameraNotAllowed)
	fmt.Printf("  URI: %s\n  Type: Boolean, Value: true  (Corp VPN always-on)\n\n",
		vpnv2.URIProfileNameAlwaysOn("Corp"))

	// --- 2. Operate on a custom URI the SDK does not know ------------------
	// A hand-written URI for a hypothetical OEM CSP. The client.Client verbs
	// accept any URI, on any transport.
	const customURI = "./Device/Vendor/OEM/ContosoAgent/TelemetryLevel"

	rec := syncml.NewRecorder()
	ctx := context.Background()
	if err := rec.Replace(ctx, customURI, client.Int(2)); err != nil {
		log.Fatalf("queue custom replace: %v", err)
	}
	if err := rec.Exec(ctx, "./Device/Vendor/OEM/ContosoAgent/Restart", client.Null()); err != nil {
		log.Fatalf("queue custom exec: %v", err)
	}

	fmt.Println("SyncML batch for the custom URIs:")
	fmt.Println(rec.Document())

	// The same raw calls work on any transport — a live syncml.Executor
	// would deliver them and parse the device's response:
	//
	//	exec := syncml.NewExecutor(mySender)
	//	v, err := exec.Get(ctx, customURI)   // typed client.Value back
}
