// Manage VPNv2 profiles end to end: profiles are dynamic (runtime-named)
// nodes, so the generated methods take the profile name as a parameter and
// the parent node gets List. Uses the in-memory fake transport.
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
	vpn := c.CSP.VPNv2

	// Create the profile container node, then set leaves beneath it.
	if err := vpn.CreateProfileName(ctx, "CorpVPN"); err != nil {
		log.Fatalf("create profile: %v", err)
	}
	if err := vpn.UpdateProfileNameRememberCredentials(ctx, "CorpVPN", true); err != nil {
		log.Fatalf("set RememberCredentials: %v", err)
	}
	if err := vpn.UpdateProfileNameAlwaysOn(ctx, "CorpVPN", true); err != nil {
		log.Fatalf("set AlwaysOn: %v", err)
	}

	profiles, err := vpn.List(ctx)
	if err != nil {
		log.Fatalf("list profiles: %v", err)
	}
	fmt.Printf("profiles: %v\n", profiles)

	on, err := vpn.GetProfileNameAlwaysOn(ctx, "CorpVPN")
	if err != nil {
		log.Fatalf("read AlwaysOn: %v", err)
	}
	fmt.Printf("CorpVPN AlwaysOn = %v\n", on)

	// Deleting the profile node removes the whole subtree.
	if err := vpn.DeleteProfileName(ctx, "CorpVPN"); err != nil {
		log.Fatalf("delete profile: %v", err)
	}
	profiles, _ = vpn.List(ctx)
	fmt.Printf("profiles after delete: %v\n", profiles)
}
