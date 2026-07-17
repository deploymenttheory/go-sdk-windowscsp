// Enumerate the MDM enrollments (providers) a device reports through the
// DMClient CSP and read per-enrollment values. Providers are dynamic nodes
// named by their enrollment ID, so the fake transport is seeded with what a
// managed device would hold.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/client"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/clienttest"
)

func main() {
	mock := clienttest.New()
	mock.Seed("./Device/Vendor/MSFT/DMClient/Provider/MS DM Server/EntDeviceName", client.Chr("LAPTOP-042"))
	mock.Seed("./Device/Vendor/MSFT/DMClient/Provider/MS DM Server/PublisherDeviceID", client.Chr("A1B2C3"))

	c := windowscsp.NewClient(mock)
	ctx := context.Background()

	providers, err := c.CSP.DMClient.ListProvider(ctx)
	if err != nil {
		log.Fatalf("list providers: %v", err)
	}
	for _, id := range providers {
		name, err := c.CSP.DMClient.GetProviderProviderIDEntDeviceName(ctx, id)
		if err != nil {
			log.Fatalf("read EntDeviceName for %q: %v", id, err)
		}
		fmt.Printf("enrollment %q: device name %s\n", id, name)
	}
}
