package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/deploymenttheory/go-sdk-windowscsp/internal/ddf"
)

const miniDDF = `<?xml version="1.0" encoding="UTF-8"?>
<MgmtTree xmlns:MSFT="http://schemas.microsoft.com/MobileDevice/DM">
  <VerDTD>1.2</VerDTD>
  <Node>
    <NodeName>Mini</NodeName>
    <Path>./Device/Vendor/MSFT</Path>
    <DFProperties>
      <AccessType><Get /></AccessType>
      <DFFormat><node /></DFFormat>
    </DFProperties>
    <Node>
      <NodeName>Value</NodeName>
      <DFProperties>
        <AccessType><Get /><Replace /></AccessType>
        <DFFormat><int /></DFFormat>
      </DFProperties>
    </Node>
  </Node>
</MgmtTree>`

func buildZip(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestRunRoundTrip(t *testing.T) {
	dir := t.TempDir()
	zipBytes := buildZip(t, map[string]string{
		"Drop/Mini.xml":   miniDDF,
		"Drop/readme.txt": "not xml",
	})
	zipPath := filepath.Join(dir, "drop.zip")
	if err := os.WriteFile(zipPath, zipBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")

	// A stale file from a previous, larger drop must be pruned.
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(outDir, "Gone.json")
	if err := os.WriteFile(stale, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := run("Test Release", "https://example.invalid/drop.zip", zipPath, "", "2026-07-17", outDir); err != nil {
		t.Fatalf("run: %v", err)
	}

	var csp ddf.CSP
	body, err := os.ReadFile(filepath.Join(outDir, "Mini.json"))
	if err != nil {
		t.Fatalf("snapshot missing: %v", err)
	}
	if err := json.Unmarshal(body, &csp); err != nil {
		t.Fatal(err)
	}
	if csp.Name != "Mini" || len(csp.Nodes) != 1 || csp.Nodes[0].Name != "Value" {
		t.Fatalf("snapshot content = %+v", csp)
	}

	var prov ddf.Provenance
	body, err = os.ReadFile(filepath.Join(outDir, "PROVENANCE.json"))
	if err != nil {
		t.Fatalf("provenance missing: %v", err)
	}
	if err := json.Unmarshal(body, &prov); err != nil {
		t.Fatal(err)
	}
	if prov.Release != "Test Release" || prov.Fetched != "2026-07-17" || prov.SHA256 == "" {
		t.Fatalf("provenance = %+v", prov)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale snapshot not pruned: %v", err)
	}
}

func TestRunSHA256Mismatch(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "drop.zip")
	if err := os.WriteFile(zipPath, buildZip(t, map[string]string{"Mini.xml": miniDDF}), 0o644); err != nil {
		t.Fatal(err)
	}
	err := run("r", "u", zipPath, "deadbeef", "2026-07-17", filepath.Join(dir, "out"))
	if err == nil {
		t.Fatal("expected sha256 mismatch error")
	}
}

func TestSnapshotName(t *testing.T) {
	cases := map[string]string{
		`DDFDrop012026\LAPS.xml`:           "LAPS",
		"DDFDrop012026/Camera_AreaDDF.xml": "Camera_AreaDDF",
		"Plain.xml":                        "Plain",
	}
	for in, want := range cases {
		if got := snapshotName(in); got != want {
			t.Errorf("snapshotName(%q) = %q, want %q", in, got, want)
		}
	}
}

const dualScopeDDF = `<?xml version="1.0" encoding="UTF-8"?>
<MgmtTree xmlns:MSFT="http://schemas.microsoft.com/MobileDevice/DM">
  <VerDTD>1.2</VerDTD>
  <Node>
    <NodeName>Dual</NodeName>
    <Path>./User/Vendor/MSFT</Path>
    <DFProperties><AccessType><Get /></AccessType><DFFormat><node /></DFFormat></DFProperties>
    <Node>
      <NodeName>V</NodeName>
      <DFProperties><AccessType><Get /></AccessType><DFFormat><int /></DFFormat></DFProperties>
    </Node>
  </Node>
  <Node>
    <NodeName>Dual</NodeName>
    <Path>./Device/Vendor/MSFT</Path>
    <DFProperties><AccessType><Get /></AccessType><DFFormat><node /></DFFormat></DFProperties>
    <Node>
      <NodeName>V</NodeName>
      <DFProperties><AccessType><Get /></AccessType><DFFormat><int /></DFFormat></DFProperties>
    </Node>
  </Node>
</MgmtTree>`

// TestParseZipSplitsScopes: dual-scope DDF files become one snapshot per
// scope, with the Device tree keeping the plain name.
func TestParseZipSplitsScopes(t *testing.T) {
	zipBytes := buildZip(t, map[string]string{
		"Drop/Dual.xml":         dualScopeDDF,
		"Drop/Dual_AreaDDF.xml": strings.ReplaceAll(dualScopeDDF, "Vendor/MSFT", "Vendor/MSFT/Policy/Config"),
	})
	csps, err := parseZip(zipBytes)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Dual", "Dual_User", "Dual_AreaDDF", "Dual_User_AreaDDF"} {
		if _, ok := csps[want]; !ok {
			t.Errorf("missing snapshot %q (have %v)", want, keys(csps))
		}
	}
	if got := csps["Dual"].Path; got != "./Device/Vendor/MSFT" {
		t.Errorf("Dual path = %q", got)
	}
	if got := csps["Dual_User"].Path; got != "./User/Vendor/MSFT" {
		t.Errorf("Dual_User path = %q", got)
	}
}

func keys(m map[string]*ddf.CSP) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestScopedName(t *testing.T) {
	cases := []struct{ name, scope, want string }{
		{"VPNv2", "User", "VPNv2_User"},
		{"InternetExplorer_AreaDDF", "User", "InternetExplorer_User_AreaDDF"},
	}
	for _, c := range cases {
		if got := scopedName(c.name, c.scope); got != c.want {
			t.Errorf("scopedName(%q, %q) = %q, want %q", c.name, c.scope, got, c.want)
		}
	}
}

func TestDiscoverRegex(t *testing.T) {
	page := `<a href="https://download.microsoft.com/download/015bd9f5-9cca-4821-8a85-a4c5f9a5d0f2/DDFv2Feb2026.zip">DDF v2 Files, February 2026</a>`
	m := ddfZipRe.FindString(page)
	if m != "https://download.microsoft.com/download/015bd9f5-9cca-4821-8a85-a4c5f9a5d0f2/DDFv2Feb2026.zip" {
		t.Fatalf("regex match = %q", m)
	}
}
