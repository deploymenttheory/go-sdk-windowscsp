package codegen

import (
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden files from generated output")

// TestGolden runs the full generator over the fixture snapshots and compares
// every emitted file to testdata/golden. Run with -update to refresh.
func TestGolden(t *testing.T) {
	out := t.TempDir()
	if err := Run(filepath.Join("testdata", "snapshot"), out, "github.com/deploymenttheory/go-sdk-windowscsp/windowscsp"); err != nil {
		t.Fatalf("Run: %v", err)
	}

	goldenDir := filepath.Join("testdata", "golden")
	got := map[string][]byte{}
	err := filepath.WalkDir(out, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(out, path)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		got[filepath.ToSlash(rel)] = body
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("generator produced no files")
	}

	if *update {
		if err := os.RemoveAll(goldenDir); err != nil {
			t.Fatal(err)
		}
		for rel, body := range got {
			path := filepath.Join(goldenDir, filepath.FromSlash(rel)+".golden")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, body, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		t.Logf("updated %d golden files", len(got))
		return
	}

	want := map[string]bool{}
	err = filepath.WalkDir(goldenDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(goldenDir, path)
		if err != nil {
			return err
		}
		rel = strings.TrimSuffix(filepath.ToSlash(rel), ".golden")
		want[rel] = true
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		gotBody, ok := got[rel]
		if !ok {
			t.Errorf("golden file %s has no generated counterpart", rel)
			return nil
		}
		if normalize(gotBody) != normalize(body) {
			t.Errorf("generated %s differs from golden (run: go test ./internal/codegen -run TestGolden -update)", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for rel := range got {
		if !want[rel] {
			t.Errorf("generated file %s has no golden counterpart (run with -update)", rel)
		}
	}
}

func normalize(b []byte) string {
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}
