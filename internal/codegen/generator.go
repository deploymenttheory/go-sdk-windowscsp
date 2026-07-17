// Package codegen orchestrates the offline CSP code generator: it loads the
// committed DDF snapshots, clears previously generated output (identified
// by the DO-NOT-EDIT header), builds view models (build), renders them
// through the template firewall (render) and assembles files (fileasm).
// Everything is deterministic from metadata/csp.
package codegen

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/deploymenttheory/go-sdk-windowscsp/internal/codegen/build"
	"github.com/deploymenttheory/go-sdk-windowscsp/internal/codegen/fileasm"
	"github.com/deploymenttheory/go-sdk-windowscsp/internal/codegen/naming"
	"github.com/deploymenttheory/go-sdk-windowscsp/internal/codegen/render"
	"github.com/deploymenttheory/go-sdk-windowscsp/internal/codegen/view"
	"github.com/deploymenttheory/go-sdk-windowscsp/internal/ddf"
)

// Run generates the SDK surface from metadataDir into outDir. modulePath is
// the import path of outDir's package tree (the windowscsp package).
func Run(metadataDir, outDir, modulePath string) error {
	release, err := loadRelease(metadataDir)
	if err != nil {
		return err
	}
	pkgs, err := loadPackages(metadataDir, release)
	if err != nil {
		return err
	}

	// Clear before re-emitting: a run never mixes fresh output with
	// leftovers from a previous generation. Only header-marked files are
	// removed; hand-written packages under the output root survive.
	if err := clearGenerated(outDir); err != nil {
		return fmt.Errorf("clear %s: %w", outDir, err)
	}
	for _, p := range pkgs {
		if err := emitPackage(outDir, p, modulePath); err != nil {
			return fmt.Errorf("%s: %w", p.Dir, err)
		}
	}
	if err := emitRegistry(outDir, pkgs, modulePath, release); err != nil {
		return err
	}
	fmt.Printf("generated %d CSP packages -> %s\n", len(pkgs), outDir)
	return nil
}

func loadRelease(metadataDir string) (string, error) {
	body, err := os.ReadFile(filepath.Join(metadataDir, "PROVENANCE.json"))
	if err != nil {
		return "", fmt.Errorf("read provenance: %w", err)
	}
	var prov ddf.Provenance
	if err := json.Unmarshal(body, &prov); err != nil {
		return "", fmt.Errorf("parse provenance: %w", err)
	}
	if prov.Release == "" {
		return "", fmt.Errorf("provenance has no release label")
	}
	return prov.Release, nil
}

func loadPackages(metadataDir, release string) ([]*view.Package, error) {
	entries, err := filepath.Glob(filepath.Join(metadataDir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(entries)

	var pkgs []*view.Package
	byDir := map[string]string{}
	for _, path := range entries {
		base := strings.TrimSuffix(filepath.Base(path), ".json")
		if base == "PROVENANCE" {
			continue
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var csp ddf.CSP
		if err := json.Unmarshal(body, &csp); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		p := build.Package(base, &csp, release)
		if p == nil {
			continue
		}
		if prev, clash := byDir[p.Dir]; clash {
			return nil, fmt.Errorf("package dir collision: %s and %s both map to %s", prev, base, p.Dir)
		}
		byDir[p.Dir] = base
		pkgs = append(pkgs, p)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no CSP snapshots in %s", metadataDir)
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Dir < pkgs[j].Dir })
	return pkgs, nil
}

func emitPackage(outDir string, p *view.Package, modulePath string) error {
	dir := filepath.Join(outDir, filepath.FromSlash(p.Dir))
	clientImport := fileasm.Import{Path: modulePath + "/client"}

	// doc.go
	doc, err := render.Doc(p)
	if err != nil {
		return err
	}
	if err := writeGen(dir, "doc.go", p.PackageName, doc, nil, ""); err != nil {
		return err
	}

	// <pkg>_service.go
	service, err := render.Service(p)
	if err != nil {
		return err
	}
	if err := writeGen(dir, p.PackageName+"_service.go", p.PackageName, "", []fileasm.Import{clientImport}, service); err != nil {
		return err
	}

	// <pkg>_uris.go
	var uris strings.Builder
	if hasConstURIs(p) {
		s, err := render.URIConsts(p)
		if err != nil {
			return err
		}
		uris.WriteString(s)
	}
	for i := range p.URIs {
		if !p.URIs[i].IsFunc {
			continue
		}
		s, err := render.URIFunc(&p.URIs[i])
		if err != nil {
			return err
		}
		uris.WriteString(s)
	}
	if uris.Len() > 0 {
		if err := writeGen(dir, p.PackageName+"_uris.go", p.PackageName, "", nil, uris.String()); err != nil {
			return err
		}
	}

	// <pkg>_crud.go
	var crud strings.Builder
	for i := range p.Methods {
		s, err := render.Method(&p.Methods[i])
		if err != nil {
			return err
		}
		crud.WriteString(s)
	}
	crudImports := []fileasm.Import{{Path: "context"}}
	if strings.Contains(crud.String(), "client.") {
		crudImports = append(crudImports, clientImport)
	}
	if err := writeGen(dir, p.PackageName+"_crud.go", p.PackageName, "", crudImports, crud.String()); err != nil {
		return err
	}

	// <pkg>_enums.go
	if len(p.Enums) > 0 {
		var enums strings.Builder
		for i := range p.Enums {
			s, err := render.EnumBlock(&p.Enums[i])
			if err != nil {
				return err
			}
			enums.WriteString(s)
		}
		var enumImports []fileasm.Import
		if strings.Contains(enums.String(), "fmt.") {
			enumImports = append(enumImports, fileasm.Import{Path: "fmt"})
		}
		if err := writeGen(dir, p.PackageName+"_enums.go", p.PackageName, "", enumImports, enums.String()); err != nil {
			return err
		}
	}
	return nil
}

func hasConstURIs(p *view.Package) bool {
	for _, u := range p.URIs {
		if !u.IsFunc {
			return true
		}
	}
	return false
}

func emitRegistry(outDir string, pkgs []*view.Package, modulePath, release string) error {
	reg := view.Registry{Release: release}
	families := []struct {
		field, typeName, doc, dir string
	}{
		{"CSP", "CSPServices", "groups the standalone configuration service providers.", "csp"},
		{"Policy", "PolicyServices", "groups the Policy CSP areas (./Vendor/MSFT/Policy/Config).", "policy"},
	}

	imports := []fileasm.Import{{Path: modulePath + "/client"}}
	seenField := map[string]string{}
	for _, fam := range families {
		f := view.Family{TypeName: fam.typeName, FieldName: fam.field, Doc: fam.doc}
		for _, p := range pkgs {
			if !strings.HasPrefix(p.Dir, fam.dir+"/") {
				continue
			}
			key := fam.dir + "." + p.ServiceName
			if prev, clash := seenField[key]; clash {
				return fmt.Errorf("service name collision in %s family: %s and %s", fam.dir, prev, p.Dir)
			}
			seenField[key] = p.Dir
			alias := naming.ImportAlias(strings.ReplaceAll(fam.dir, "/", ""), p.PackageName)
			f.Entries = append(f.Entries, view.FamilyEntry{
				FieldName:   p.ServiceName,
				ServiceName: p.ServiceName,
				Alias:       alias,
				ImportPath:  modulePath + "/" + p.Dir,
			})
			imports = append(imports, fileasm.Import{Alias: alias, Path: modulePath + "/" + p.Dir})
		}
		sort.Slice(f.Entries, func(i, j int) bool { return f.Entries[i].FieldName < f.Entries[j].FieldName })
		reg.Families = append(reg.Families, f)
	}

	body, err := render.Registry(&reg)
	if err != nil {
		return err
	}
	return writeGen(outDir, "registry.go", "windowscsp", "", imports, body)
}

func writeGen(dir, name, pkg, docComment string, imports []fileasm.Import, body string) error {
	return fileasm.WriteFile(filepath.Join(dir, name), pkg, docComment, imports, body)
}

// clearGenerated removes every generated file (identified by the header
// marker) under outDir, then removes emptied directories. Hand-written
// files are never touched.
func clearGenerated(outDir string) error {
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		return nil
	}
	var dirs []string
	err := filepath.WalkDir(outDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		head := make([]byte, len(fileasm.Header))
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		n, _ := f.Read(head)
		f.Close()
		if string(head[:n]) == fileasm.Header {
			return os.Remove(path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	// Remove emptied directories, deepest first.
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err == nil && len(entries) == 0 {
			os.Remove(dir)
		}
	}
	return nil
}
