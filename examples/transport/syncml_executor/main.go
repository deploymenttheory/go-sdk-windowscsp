// SyncML handled transparently: syncml.Executor renders each SDK call to
// SyncML, delivers it through a SenderFunc you supply, and parses the
// device's SyncML response back into typed values and errors. The calling
// code below never touches SyncML — reads included.
//
// The SenderFunc here simulates a device so the example runs offline; in
// production it would hand the request to your OMA-DM session or MDM proxy
// and return the device's reply.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/policy/camera"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/syncml"
)

// fakeDevice answers every command with status 200 and every Get with an
// int value of 1, standing in for a real OMA-DM endpoint.
func fakeDevice(_ context.Context, request string) (string, error) {
	fmt.Println("--- SyncML sent to device ---")
	fmt.Print(request)

	return `<SyncBody>
	  <Status><CmdID>1</CmdID><CmdRef>1</CmdRef><Data>200</Data></Status>
	  <Results><CmdID>2</CmdID><CmdRef>1</CmdRef>
	    <Item>
	      <Source><LocURI>` + camera.URIAllowCamera + `</LocURI></Source>
	      <Meta><Format xmlns="syncml:metinf">int</Format></Meta>
	      <Data>1</Data>
	    </Item>
	  </Results>
	</SyncBody>`, nil
}

func main() {
	c := windowscsp.NewClient(syncml.NewExecutor(fakeDevice))
	ctx := context.Background()

	// Plain typed calls; SyncML request/response handling is invisible.
	if err := c.Policy.Camera.UpdateAllowCamera(ctx, camera.AllowCameraNotAllowed); err != nil {
		log.Fatalf("update: %v", err)
	}
	allow, err := c.Policy.Camera.GetAllowCamera(ctx)
	if err != nil {
		log.Fatalf("get: %v", err)
	}
	fmt.Printf("--- typed result ---\nAllowCamera = %d\n", allow)
}
