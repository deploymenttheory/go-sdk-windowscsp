// Configure Windows LAPS policies, trigger an immediate password reset and
// poll the action's status node. Uses the in-memory fake transport.
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
	c := windowscsp.NewClient(mock)
	ctx := context.Background()
	laps := c.CSP.LAPS

	// Policy configuration: manage the built-in Administrator account with
	// a 20-character password.
	if err := laps.UpdatePoliciesAdministratorAccountName(ctx, "Administrator"); err != nil {
		log.Fatalf("set account name: %v", err)
	}
	if err := laps.UpdatePoliciesPasswordLength(ctx, 20); err != nil {
		log.Fatalf("set password length: %v", err)
	}

	// Trigger an immediate rotation.
	if err := laps.ExecActionsResetPassword(ctx); err != nil {
		log.Fatalf("exec ResetPassword: %v", err)
	}

	// On a real device the client reports progress here; the fake transport
	// lets us seed what a device would answer.
	mock.Seed("./Device/Vendor/MSFT/LAPS/Actions/ResetPasswordStatus", client.Int(1))
	status, err := laps.GetActionsResetPasswordStatus(ctx)
	if err != nil {
		log.Fatalf("read status: %v", err)
	}
	fmt.Printf("ResetPasswordStatus = %d (1 = success)\n", status)
}
