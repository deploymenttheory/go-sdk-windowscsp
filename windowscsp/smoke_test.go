package windowscsp_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/clienttest"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/policy/camera"
	"github.com/deploymenttheory/go-sdk-windowscsp/windowscsp/syncml"
)

// These tests exercise the generated surface end to end: registry wiring,
// URI construction, typed value round-trips and dynamic-node parameters,
// against the in-memory fake transport.

func TestSmokeRebootExecAndSchedule(t *testing.T) {
	mock := clienttest.New()
	c := windowscsp.NewClient(mock)
	ctx := context.Background()

	if err := c.CSP.Reboot.ExecRebootNow(ctx); err != nil {
		t.Fatal(err)
	}
	if len(mock.Executed) != 1 || mock.Executed[0].URI != "./Device/Vendor/MSFT/Reboot/RebootNow" {
		t.Fatalf("Executed = %+v", mock.Executed)
	}

	if err := c.CSP.Reboot.UpdateScheduleSingle(ctx, "2026-08-01T03:00:00"); err != nil {
		t.Fatal(err)
	}
	got, err := c.CSP.Reboot.GetScheduleSingle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "2026-08-01T03:00:00" {
		t.Fatalf("GetScheduleSingle = %q", got)
	}
}

func TestSmokePolicyCameraEnum(t *testing.T) {
	mock := clienttest.New()
	c := windowscsp.NewClient(mock)
	ctx := context.Background()

	if err := c.Policy.Camera.UpdateAllowCamera(ctx, camera.AllowCameraNotAllowed); err != nil {
		t.Fatal(err)
	}
	got, err := c.Policy.Camera.GetAllowCamera(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != camera.AllowCameraNotAllowed {
		t.Fatalf("GetAllowCamera = %d", got)
	}
	if camera.URIAllowCamera != "./Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera" {
		t.Fatalf("URIAllowCamera = %q", camera.URIAllowCamera)
	}
}

func TestSmokeVPNv2DynamicProfiles(t *testing.T) {
	mock := clienttest.New()
	c := windowscsp.NewClient(mock)
	ctx := context.Background()

	if err := c.CSP.VPNv2.CreateProfileNameRememberCredentials(ctx, "Corp", true); err != nil {
		t.Fatal(err)
	}
	on, err := c.CSP.VPNv2.GetProfileNameRememberCredentials(ctx, "Corp")
	if err != nil {
		t.Fatal(err)
	}
	if !on {
		t.Fatal("RememberCredentials round-trip lost the value")
	}

	names, err := c.CSP.VPNv2.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(names, []string{"Corp"}) {
		t.Fatalf("List = %v", names)
	}

	if err := c.CSP.VPNv2.DeleteProfileName(ctx, "Corp"); err != nil {
		t.Fatal(err)
	}
	names, _ = c.CSP.VPNv2.List(ctx)
	if len(names) != 0 {
		t.Fatalf("List after delete = %v", names)
	}
}

func TestSmokeSyncMLRecorder(t *testing.T) {
	rec := syncml.NewRecorder()
	c := windowscsp.NewClient(rec)
	ctx := context.Background()

	if err := c.Policy.Camera.UpdateAllowCamera(ctx, 0); err != nil {
		t.Fatal(err)
	}
	if err := c.CSP.Reboot.ExecRebootNow(ctx); err != nil {
		t.Fatal(err)
	}

	doc := rec.Document()
	for _, want := range []string{
		"<Replace>",
		"<LocURI>./Device/Vendor/MSFT/Policy/Config/Camera/AllowCamera</LocURI>",
		`<Format xmlns="syncml:metinf">int</Format>`,
		"<Data>0</Data>",
		"<Exec>",
		"<LocURI>./Device/Vendor/MSFT/Reboot/RebootNow</LocURI>",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("SyncML document missing %q:\n%s", want, doc)
		}
	}
}

func TestSmokeRegistryWiring(t *testing.T) {
	c := windowscsp.NewClient(clienttest.New())
	if c.CSP == nil || c.Policy == nil {
		t.Fatal("families not wired")
	}
	if c.CSP.Reboot == nil || c.CSP.LAPS == nil || c.Policy.Camera == nil {
		t.Fatal("services not wired")
	}
	if c.Transport() == nil {
		t.Fatal("transport not retained")
	}
}
