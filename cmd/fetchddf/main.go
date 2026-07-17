// Command fetchddf is the acquisition stage of the CSP pipeline. It downloads
// a pinned, versioned DDF v2 zip (Microsoft's canonical, machine-readable CSP
// schema), verifies its SHA-256, parses every CSP/policy area, and writes the
// committed snapshot tree (metadata/csp/<name>.json) plus provenance.
//
// Codegen (cmd/gencsp) is offline and deterministic from the committed
// snapshots; fetching a fresh DDF release is a deliberate, reviewed act.
//
//	go run ./cmd/fetchddf                     # download the pinned release
//	go run ./cmd/fetchddf -zip local.zip      # parse an already-downloaded zip (offline)
//	go run ./cmd/fetchddf -discover           # find the newest drop on Microsoft Learn
//	go run ./cmd/fetchddf -url <u> -release <r> -sha256 <hex>
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/deploymenttheory/go-sdk-windowscsp/internal/ddf"
)

// Pinned DDF v2 release. Bump these (and re-run) to move to a new drop.
const (
	defaultRelease = "February 2026"
	defaultURL     = "https://download.microsoft.com/download/015bd9f5-9cca-4821-8a85-a4c5f9a5d0f2/DDFv2Feb2026.zip"
	defaultSHA256  = "bf667d895af4a8c8ab5a31065ce0e28ea2f8b649c4dc416f452f62fd1c42ff14"

	// ddfDocURL is the Microsoft Learn page listing DDF v2 downloads, used by
	// -discover to find drops newer than the pin.
	ddfDocURL = "https://learn.microsoft.com/en-us/windows/client-management/mdm/configuration-service-provider-ddf"
)

func main() {
	release := flag.String("release", defaultRelease, "DDF release label recorded in provenance")
	url := flag.String("url", defaultURL, "DDF v2 zip URL")
	zipPath := flag.String("zip", "", "parse a local zip instead of downloading (offline)")
	wantSHA := flag.String("sha256", defaultSHA256, "expected zip SHA-256 (hex); verified when set, pass '' to skip")
	fetched := flag.String("fetched", "", "fetch date YYYY-MM-DD recorded in provenance (default: today UTC)")
	outDir := flag.String("out", filepath.Join("metadata", "csp"), "snapshot output directory")
	discover := flag.Bool("discover", false, "scrape the Microsoft Learn DDF page for the current drop URL; overrides -url/-release/-sha256 when a different drop is found")
	flag.Parse()

	if *fetched == "" {
		*fetched = time.Now().UTC().Format("2006-01-02")
	}
	if *discover {
		u, r, err := discoverLatest(ddfDocURL)
		if err != nil {
			fmt.Fprintln(os.Stderr, "fetchddf: discover:", err)
			os.Exit(1)
		}
		if u != *url {
			fmt.Printf("discovered new drop: %s (%s)\n", u, r)
			*url, *release, *wantSHA = u, r, ""
		} else {
			fmt.Printf("pinned drop is current: %s\n", u)
		}
	}
	if err := run(*release, *url, *zipPath, *wantSHA, *fetched, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, "fetchddf:", err)
		os.Exit(1)
	}
}

func run(release, url, zipPath, wantSHA, fetched, outDir string) error {
	data, source, err := acquire(url, zipPath)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	if wantSHA != "" && !strings.EqualFold(wantSHA, digest) {
		return fmt.Errorf("sha256 mismatch: got %s, want %s", digest, wantSHA)
	}

	csps, err := parseZip(data)
	if err != nil {
		return err
	}
	if len(csps) == 0 {
		return fmt.Errorf("no DDF (.xml) entries in the zip")
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	written := map[string]bool{}
	names := make([]string, 0, len(csps))
	for name := range csps {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(outDir, name+".json")
		body, err := json.MarshalIndent(csps[name], "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, append(body, '\n'), 0o644); err != nil {
			return err
		}
		written[path] = true
	}

	prov := ddf.Provenance{Release: release, Source: source, SHA256: digest, Fetched: fetched}
	provBody, err := json.MarshalIndent(prov, "", "  ")
	if err != nil {
		return err
	}
	provPath := filepath.Join(outDir, "PROVENANCE.json")
	if err := os.WriteFile(provPath, append(provBody, '\n'), 0o644); err != nil {
		return err
	}
	written[provPath] = true

	if err := pruneStale(outDir, written); err != nil {
		return err
	}
	fmt.Printf("fetched %s (%s): %d CSPs -> %s\n", release, digest[:12], len(csps), outDir)
	return nil
}

// acquire returns the zip bytes and a source label, from a local file or by
// downloading.
func acquire(url, zipPath string) (data []byte, source string, err error) {
	if zipPath != "" {
		data, err = os.ReadFile(zipPath)
		return data, url, err
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, url, fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, url, fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}
	data, err = io.ReadAll(resp.Body)
	return data, url, err
}

// parseZip parses every .xml entry, keyed by the entry base name. A DDF
// file may define several root trees (Device and User scopes of the same
// CSP); each becomes its own snapshot, with non-Device scopes suffixed
// into the name ("VPNv2_User", "InternetExplorer_User_AreaDDF").
func parseZip(data []byte) (map[string]*ddf.CSP, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	out := map[string]*ddf.CSP{}
	for _, entry := range zr.File {
		if entry.FileInfo().IsDir() || !strings.EqualFold(filepath.Ext(entry.Name), ".xml") {
			continue
		}
		rc, err := entry.Open()
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		csps, err := ddf.Parse(body)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name, err)
		}
		for _, csp := range csps {
			name := snapshotName(entry.Name)
			if len(csps) > 1 {
				if scope := pathScope(csp.Path); scope != "Device" {
					name = scopedName(name, scope)
				}
			}
			if _, clash := out[name]; clash {
				return nil, fmt.Errorf("%s: snapshot name collision: %q", entry.Name, name)
			}
			out[name] = csp
		}
	}
	return out, nil
}

// pathScope extracts the enrollment scope from a root path:
// "./User/Vendor/MSFT" -> "User", "./Device/Vendor/MSFT" -> "Device".
func pathScope(path string) string {
	scope := strings.TrimPrefix(path, "./")
	if i := strings.IndexByte(scope, '/'); i >= 0 {
		scope = scope[:i]
	}
	return scope
}

// scopedName splices a scope into a snapshot name, keeping the _AreaDDF
// marker (which drives Policy-area package naming) as the suffix.
func scopedName(name, scope string) string {
	if base, ok := strings.CutSuffix(name, "_AreaDDF"); ok {
		return base + "_" + scope + "_AreaDDF"
	}
	return name + "_" + scope
}

// snapshotName derives a stable snapshot file base from a DDF entry name,
// dropping the path and .xml extension.
func snapshotName(entry string) string {
	base := filepath.Base(strings.ReplaceAll(entry, `\`, "/"))
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// pruneStale removes snapshot files this run did not write (a shrinking DDF
// drop), leaving non-JSON files alone.
func pruneStale(outDir string, written map[string]bool) error {
	entries, err := filepath.Glob(filepath.Join(outDir, "*.json"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !written[entry] {
			if err := os.Remove(entry); err != nil {
				return err
			}
		}
	}
	return nil
}

var ddfZipRe = regexp.MustCompile(`https://download\.microsoft\.com/[^"'()\s]+/DDFv2[A-Za-z0-9]+\.zip`)

// discoverLatest fetches the Microsoft Learn DDF page and returns the first
// (current) DDF v2 zip URL plus a release label derived from its file name.
func discoverLatest(docURL string) (url, release string, err error) {
	resp, err := http.Get(docURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GET %s: HTTP %d", docURL, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", "", err
	}
	m := ddfZipRe.Find(body)
	if m == nil {
		return "", "", fmt.Errorf("no DDFv2 zip link found on %s", docURL)
	}
	url = string(m)
	base := strings.TrimSuffix(filepath.Base(url), ".zip")
	return url, strings.TrimPrefix(base, "DDFv2"), nil
}
