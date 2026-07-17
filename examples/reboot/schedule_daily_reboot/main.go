// Configure a daily reboot schedule and read it back. Uses the in-memory
// fake transport so the round-trip is observable without a device.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/clienttest"
)

func main() {
	c := windowscsp.NewClient(clienttest.New())
	ctx := context.Background()

	// ISO 8601 date and time; the reboot recurs daily at this time.
	const schedule = "2026-08-01T03:00:00"
	if err := c.CSP.Reboot.UpdateScheduleDailyRecurrent(ctx, schedule); err != nil {
		log.Fatalf("set schedule: %v", err)
	}

	got, err := c.CSP.Reboot.GetScheduleDailyRecurrent(ctx)
	if err != nil {
		log.Fatalf("read schedule: %v", err)
	}
	fmt.Printf("daily reboot scheduled for %s\n", got)

	// Clearing the schedule is a delete of the same node.
	if err := c.CSP.Reboot.DeleteScheduleDailyRecurrent(ctx); err != nil {
		log.Fatalf("clear schedule: %v", err)
	}
	fmt.Println("schedule cleared")
}
